// Package confy requires the following variables set in the environment to work:
// * VAULT_ADDR - The Vault server address to use
// * VAULT_AUTH_METHOD - Only "jwt" is supported in a kubernetes environment. If anything
// else is set in this value, the login token will be read from $HOME/.vault-token.
// * VAULT_PATH - The Vault role path to use.
// * VAULT_ROLE - The Vault role to use for getting an auth token.
// * VAULT_CLIENT_TIMEOUT - The client timeout to use when sending requests to Vault. This is
// optional, since the client uses a default of 60 seconds.
//
// confy uses a cache to avoid going to Vault on every value fetch. You can set the TTL of
// the cache values when you call New().
package confy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/bank-vaults/vault-sdk/vault"
	"github.com/jellydator/ttlcache/v3"
)

const (
	DefaultCacheTTL = 5 * time.Minute
	MinimumCacheTTL = 30 * time.Second	
)

var (
	replacer = strings.NewReplacer("/", "_", "#", "_")
)

// NewVaultClient is a helper method to create a vault client that
// the configuration can use.
func NewVaultClient(opts ...vault.ClientOption) *vault.Client {
	clientOptions := []vault.ClientOption{}
	clientOptions = append(clientOptions, opts...)

	// Support either local or kubernetes based authentication
	if os.Getenv("VAULT_AUTH_METHOD") != "jwt" {
		if os.Getenv("VAULT_TOKEN") != "" {
			clientOptions = append(clientOptions, vault.ClientToken(os.Getenv("VAULT_TOKEN")))
		} else {
			clientOptions = append(clientOptions, vault.ClientTokenPath(os.Getenv("HOME")+"/.vault-token"))
		}
	} else {
		clientOptions = append(clientOptions,
			vault.ClientRole(os.Getenv("VAULT_ROLE")),
			vault.ClientAuthPath(os.Getenv("VAULT_PATH")),
			vault.ClientAuthMethod(os.Getenv("VAULT_AUTH_METHOD")),
		)
	}

	client, err := vault.NewClientWithOptions(clientOptions...)
	if err != nil {
		panic(err)
	}

	return client
}

type Confy interface {
	// Get will fetch the path from Vault.
	// The path is in the format of a slash delimited string
	// and uses the pound symbol to indicate a field name.
	// If no field name is provided, the whole data document
	// is returned as a value. You will be able to invoke Data()
	// on the value in that case.
	// Example: "scylladb/app#user"
	// The value will be fetched from the secret/scylladb/app, and
	// the value of the field "user" will be returned.
	//
	// If the configuration was instantiated with envOverride==true,
	// then it will first check the environment for the value.
	// It does the lookup by upper-casing the path, and replacing any
	// slashes and pound characters with underscores. If this lookup
	// fails (i.e. returns nothing), then it will go on to lookup the
	// value in Vault.
	Get(ctx context.Context, path string) (Value, error)
	// GetOrDefault accepts a default value as a second parameter.
	// It wraps around the Get method.
	// If retrieving the value from Vault fails, it will fallback
	// to the provided value. The second return value will indicate if the value
	// is from Vault (true; which could mean it was overridden from the environment
	// if envOverride==true), or the provided fallback value (false).
	GetOrDefault(ctx context.Context, path, fallback string) (Value, bool)
	// Watch will poll to check if a value has changed. You have to provide the compare function
	// and the callback that gets called if the compare function returns true.
	// It returns a cancel function that stops the watch if called.
	Watch(path string, comparator func(oldval, newval Value) bool, callback func(v Value)) context.CancelFunc
	// Close will stop the internal automatic expiration of items from within the cache and the automatic
	// token renewal. Call it once you are done with the configuration client.
	Close()
}

type Value interface {
	// Raw returns the raw field value as received from Vault.
	Raw() any

	// These methods try to coerce the value to the requested type
	// if it does not type assert to it. If the value can't be coerced, you will
	// get the zero value of the type returned and a boolean indicating if coercion
	// failed (false).

	// Data returns the raw data map received from vault. Only works if you did
	// not specify a field name in the path.
	Data() (map[string]any, bool)
	Bool() (bool, bool)
	Float64() (float64, bool)
	Int64() (int64, bool)
	Int() (int, bool)
	Map() (map[string]string, bool)
	StringSlice() ([]string, bool)
	String() string
	Duration() (time.Duration, bool)
}

// New will return a configuration client that can be used to fetch values from
// Vault. envOverride will make the Get calls first check for the value in the environment.
// cacheTTL specifies how long will the values be cached in memory, before re-fetching them
// from Vault.
// You should call Close() on the object it returns once you are done with it to stop the internal
// expiration of items in its cache and the automatic token renewal.
//
// Passing a cacheTTL of 0 will cause the DefaultCacheTTL value to be used. Also, the minimum
// allowed cacheTTL is 30 seconds. Anything less than this will cause the MinimumCacheTTL to be
// used instead.
func New(client *vault.Client, cacheTTL time.Duration, envOverride bool) Confy {
	if cacheTTL == 0 {
		cacheTTL = DefaultCacheTTL
	}

	// Avoids abusing Vault
	if cacheTTL < MinimumCacheTTL {
		cacheTTL = MinimumCacheTTL
	}

	return new(client, cacheTTL, envOverride)
}

func new(client *vault.Client, cacheTTL time.Duration, envOverride bool) Confy {
	cache := ttlcache.New(
		ttlcache.WithTTL[string, map[string]any](cacheTTL),
	)
	go cache.Start()
	return &confyImpl{cache: cache, envOverride: envOverride, client: client, ttl: cacheTTL}
}

func createLoader(ctx context.Context, c *vault.Client, e *error) ttlcache.Loader[string, map[string]any] {
	return ttlcache.NewSuppressedLoader[string, map[string]any](ttlcache.LoaderFunc[string, map[string]any](func(cache *ttlcache.Cache[string, map[string]any], key string) *ttlcache.Item[string, map[string]any] { //nolint:lll
		resp, err := c.RawClient().KVv1("secret").Get(ctx, key)
		if err != nil {
			*e = fmt.Errorf("could not get secret from Vault: %w", err)
			return nil
		}

		return cache.Set(key, resp.Data, ttlcache.DefaultTTL)
	}), nil)
}

type confyImpl struct {
	cache       *ttlcache.Cache[string, map[string]any]
	envOverride bool
	client      *vault.Client
	ttl         time.Duration
	closed      bool
}

func (c *confyImpl) Close() {
	if !c.closed {
		c.cache.Stop()
		c.client.Close()
		c.closed = true
	}
}

func (c *confyImpl) Get(ctx context.Context, path string) (Value, error) {
	path = strings.TrimPrefix(path, "secret/")
	if c.envOverride {
		envKey := strings.ToUpper(replacer.Replace(path))
		envValue := os.Getenv(envKey)
		if envValue != "" {
			return &value{val: envValue}, nil
		}
	}

	parts := strings.SplitN(path, "#", 2)
	path = parts[0]
	var fieldName string
	if len(parts) > 1 {
		fieldName = parts[1]
	}

	var errBucket error
	loader := createLoader(ctx, c.client, &errBucket)
	v := c.cache.Get(path, ttlcache.WithLoader(loader))
	if v == nil {
		if errBucket != nil {
			return nil, errBucket
		} else {
			return nil, errors.New("no value found")
		}
	}

	if fieldName != "" {
		if f, ok := v.Value()[fieldName]; ok {
			return &value{val: f}, nil
		} else {
			return nil, fmt.Errorf("field '%s' on path '%s' was not found", fieldName, path)
		}
	}

	return &value{val: v.Value()}, nil
}

func (c *confyImpl) GetOrDefault(ctx context.Context, path, fallback string) (Value, bool) {
	v, err := c.Get(ctx, path)
	if err != nil {
		return &value{val: fallback}, false
	}

	return v, true
}

type value struct {
	val any
}

func (v *value) String() string {
	if s, ok := v.val.(string); ok {
		return s
	}

	return fmt.Sprintf("%s", v.val)
}

func (v *value) Raw() any {
	return v.val
}

func (v *value) Data() (map[string]any, bool) {
	m, ok := v.val.(map[string]any)
	return m, ok
}

func (v *value) Bool() (bool, bool) {
	b, ok := v.val.(bool)
	var err error
	if !ok {
		b, err = strconv.ParseBool(fmt.Sprintf("%s", v.val))
		if err != nil {
			return false, false
		}
	}

	return b, true
}

func (v *value) Float64() (float64, bool) {
	n, ok := v.val.(json.Number)
	if !ok {
		i, err := strconv.ParseFloat(fmt.Sprintf("%s", v.val), 64)
		if err != nil {
			return 0, false
		}
		return i, true
	}

	i, err := n.Float64()
	if err != nil {
		return 0, false
	}

	return i, true
}

func (v *value) Int64() (int64, bool) {
	n, ok := v.val.(json.Number)
	if !ok {
		i, err := strconv.ParseInt(fmt.Sprintf("%s", v.val), 10, 64)
		if err != nil {
			return 0, false
		}
		return i, true
	}

	i, err := n.Int64()
	if err != nil {
		return 0, false
	}

	return i, true
}

func (v *value) Int() (int, bool) {
	i, ok := v.Int64()
	return int(i), ok
}

func (v *value) Map() (map[string]string, bool) {
	ma, ok := v.val.(map[string]any)
	if !ok {
		return map[string]string{}, false
	}

	ms := make(map[string]string, len(ma))
	for k, v := range ma {
		str, ok := v.(string)
		if !ok {
			ms[k] = fmt.Sprintf("%s", v)
		} else {
			ms[k] = str
		}
	}

	return ms, true
}

func (v *value) StringSlice() ([]string, bool) {
	vals, ok := v.val.([]any)
	if !ok {
		return []string{}, false
	}

	strs := make([]string, len(vals))
	for i, val := range vals {
		s, ok := val.(string)
		if !ok {
			strs[i] = fmt.Sprintf("%s", val)
		} else {
			strs[i] = s
		}
	}

	return strs, true
}

func (v *value) Duration() (time.Duration, bool) {
	s := v.String()
	d, err := time.ParseDuration(s)
	if err != nil {
		return time.Duration(0), false
	}

	return d, true
}

// Watch will poll to check if a value has changed. You have to provide the compare function
// and the callback that gets called if the compare function returns true.
// It returns a cancel function that stops the watch if called.
func (c *confyImpl) Watch(path string, comparator func(oldval, newval Value) bool, callback func(v Value)) context.CancelFunc {
	// start polling goroutine with select
	// return function that will push signal to kill thread
	stopChan := make(chan struct{})
	go func() {
		oldValue, err := c.Get(context.Background(), path)
		if err != nil {
			oldValue = &value{val: ""}
		}
	OuterLoop:
		for {
			select {
			case <-time.After(c.ttl + (time.Second)):
				newValue, err := c.Get(context.Background(), path)
				if err != nil {
					break
				}
				if comparator(oldValue, newValue) {
					callback(newValue)
				}
				oldValue = newValue
			case <-stopChan:
				break OuterLoop
			}
		}
	}()

	return func() {
		stopChan <- struct{}{}
	}
}
