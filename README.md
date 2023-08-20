# Confy

A dynamic configuration client that works with Vault.

## Overview

* Uses bank-vault's vault sdk (the same one used by our vault injection solution) to create a vault-based configuration client that will automatically refresh the login token. Uses the same JWT/Kubernetes method supported by Vault. 
* You can either get a specific key, or the whole document (or data map) depending on how you specify the vault path. Uses the same notation that bank-vault's injection uses. i.e. `#` delimits the field name at the given path. If you get the document, you could unmarshal this into any custom struct using `mapstructure`.
* You can configure the client to be overridden by environment variables when it tries to fetch a value. Environment name matching rules are described in the source code.
* Provides a get method that allows you to fallback to a provided default value if there is an error.
* Will cache values in memory with a configurable expiration. Caching happens at the document (or path) level, so getting multiple fields from the same vault path will benefit from the same cached document.
* Provides a watch function. You provide the callback function, and how to compare the old and new values. The polling time interval is automatically determined based on the cache TTL for ease of use.
* Comes with convenience functions for doing type conversions. Supports string, bool, floats, int64, string map, string slice, and time duration.

## Usage

**Interfaces**:
```go
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
```

**Instantiation**:
```go
// New will return a configuration client that can be used to fetch values from
// Vault. envOverride will make the Get calls first check for the value in the environment.
// cacheTTL specifies how long will the values be cached in memory, before re-fetching them
// from Vault.
// You should call Close() on the object it returns once you are done with it to stop the internal
// expiration of items in its cache and the automatic token renewal.
func New(client *vault.Client, cacheTTL time.Duration, envOverride bool) Confy
```

Install with:
```
go get github.com/renier/confy@latest
```

**Simple example**:

```go
import (
	"context"

	"github.com/renier/confy"
)

...

config := confy.New(confy.NewVaultClient(), 5 * time.Minute, false)

v, ok := config.GetOrDefault(context.Background(), "scylladb/app#user", os.Getenv("DEFAULT_SCYLLA_USER"))
if !ok {
	logger.Warn().Str("user", v.String()).Msg("using default scylla user")
}

fmt.Println(v.String())
```

To use locally, make sure you set the `VAULT_ADDR` environment variable to the server you want to use. Then, either separately login to vault (by ensuring your token gets created at `~/.vault-token`) OR put the vault token in the `VAULT_TOKEN` environment variable. That's it.

To use in kubernetes, it needs several things:
* Environment variables `VAULT_AUTH_METHOD=jwt`, `VAULT_ROLE`, `VAULT_PATH`, and `VAULT_ADDR`. These will depend on how your Vault setup is configured for jwt/oidc auth.
* This will be the default, but since it is possible to disable, ensure your container is getting a kubernetes service account token.
* If your container is running a scratch image, ensure you have a good certificate chain copied into it, so that the vault client can verify the vault server's certificate. See the `Dockerfile` for how this is done.

See `example/main.go` for more.
