package confy

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

func TestConfy(t *testing.T) {
	config := New(NewVaultClient(), 2*time.Minute, false)
	defer config.Close()
	ctx := context.Background()

	t.Run("happy path", func(t *testing.T) {
		v, err := config.Get(ctx, "test/app#user")
		if err != nil {
			t.Fatalf("%s", err)
		}

		if v.String() != "fake-user" {
			t.Fatalf("on test/app#user; expected 'fake-user'; got '%s'", v.String())
		}
	})

	t.Run("no field name", func(t *testing.T) {
		v, err := config.Get(ctx, "test/app")
		if err != nil {
			t.Fatalf("did not expect an error: %s", err)
		}

		if v == nil {
			t.Fatalf("did not expect nil")
		}
	})

	t.Run("wrong path", func(t *testing.T) {
		v, err := config.Get(ctx, "test/non-existent")
		if err == nil {
			t.Fatalf("expected an error")
		}

		if v != nil {
			t.Fatalf("expected nil")
		}
	})

	t.Run("falls back to the default", func(t *testing.T) {
		v, ok := config.GetOrDefault(ctx, "not/here#never", "foo")
		if ok {
			t.Fatalf("expected false")
		}

		if v.String() != "foo" {
			t.Fatalf("expected 'foo'")
		}
	})

	t.Run("we can get a string value", func(t *testing.T) {
		v, err := config.Get(ctx, "test/types#s")
		if err != nil {
			t.Fatalf("did not expect an error: %s", err)
		}

		expected := "a string"
		if v.String() != expected {
			t.Fatalf("expected '%s'; got '%s'", expected, v.String())
		}
	})

	t.Run("we can get a coerced string value", func(t *testing.T) {
		v, err := config.Get(ctx, "test/types#ss")
		if err != nil {
			t.Fatalf("did not expect an error: %s", err)
		}

		expected := "123"
		if v.String() != expected {
			t.Fatalf("expected '%s'; got '%s'", expected, v.String())
		}
	})

	t.Run("we can get a boolean value", func(t *testing.T) {
		v, err := config.Get(ctx, "test/types#b")
		if err != nil {
			t.Fatalf("did not expect an error: %s", err)
		}

		got, ok := v.Bool()
		if !ok {
			t.Fatalf("did not expect not being able to convert a boolean")
		}

		expected := true
		if got != expected {
			t.Fatalf("expected '%t'; got '%t'", expected, got)
		}
	})

	t.Run("we can get a coerced boolean value", func(t *testing.T) {
		v, err := config.Get(ctx, "test/types#bb")
		if err != nil {
			t.Fatalf("did not expect an error: %s", err)
		}

		got, ok := v.Bool()
		if !ok {
			t.Fatalf("did not expect not being able to coerce a boolean")
		}

		expected := true
		if got != expected {
			t.Fatalf("expected '%t'; got '%t'", expected, got)
		}
	})

	t.Run("we cannot get a coerced boolean value", func(t *testing.T) {
		v, err := config.Get(ctx, "test/types#bbb")
		if err != nil {
			t.Fatalf("did not expect an error: %s", err)
		}

		got, ok := v.Bool()
		if ok {
			t.Fatalf("did not expect being able to coerce a random string into a boolean")
		}

		expected := false
		if got != expected {
			t.Fatalf("expected '%t'; got '%t'", expected, got)
		}
	})

	t.Run("we can get an integer value", func(t *testing.T) {
		v, err := config.Get(ctx, "test/types#i")
		if err != nil {
			t.Fatalf("did not expect an error: %s", err)
		}

		got, ok := v.Int()
		if !ok {
			t.Fatalf("did not expect not being able to convert value to integer")
		}

		expected := 9
		if got != expected {
			t.Fatalf("expected '%d'; got '%d'", expected, got)
		}
	})

	t.Run("we can get a coerced integer value", func(t *testing.T) {
		v, err := config.Get(ctx, "test/types#ii")
		if err != nil {
			t.Fatalf("did not expect an error: %s", err)
		}

		got, ok := v.Int()
		if !ok {
			t.Fatalf("did not expect not being able to coerce value to an integer")
		}

		expected := 8
		if got != expected {
			t.Fatalf("expected '%d'; got '%d'", expected, got)
		}
	})

	t.Run("we cannot get a coerced integer value", func(t *testing.T) {
		v, err := config.Get(ctx, "test/types#iii")
		if err != nil {
			t.Fatalf("did not expect an error: %s", err)
		}

		got, ok := v.Int()
		if ok {
			t.Fatalf("did not expect being able to coerce value to an integer from a random string")
		}

		expected := 0
		if got != expected {
			t.Fatalf("expected '%d'; got '%d'", expected, got)
		}
	})

	t.Run("we can get an integer 64 value", func(t *testing.T) {
		v, err := config.Get(ctx, "test/types#i")
		if err != nil {
			t.Fatalf("did not expect an error: %s", err)
		}

		got, ok := v.Int64()
		if !ok {
			t.Fatalf("did not expect not being able to convert value to integer 64")
		}

		expected := int64(9)
		if got != expected {
			t.Fatalf("expected '%d'; got '%d'", expected, got)
		}
	})

	t.Run("we can get a coerced integer 64 value", func(t *testing.T) {
		v, err := config.Get(ctx, "test/types#ii")
		if err != nil {
			t.Fatalf("did not expect an error: %s", err)
		}

		got, ok := v.Int64()
		if !ok {
			t.Fatalf("did not expect not being able to coerce value to an integer 64")
		}

		expected := int64(8)
		if got != expected {
			t.Fatalf("expected '%d'; got '%d'", expected, got)
		}
	})

	t.Run("we cannot get a coerced integer 64 value", func(t *testing.T) {
		v, err := config.Get(ctx, "test/types#iii")
		if err != nil {
			t.Fatalf("did not expect an error: %s", err)
		}

		got, ok := v.Int64()
		if ok {
			t.Fatalf("did not expect being able to coerce value to an integer 64 from a random string")
		}

		expected := int64(0)
		if got != expected {
			t.Fatalf("expected '%d'; got '%d'", expected, got)
		}
	})

	t.Run("we can get an float 64 value", func(t *testing.T) {
		v, err := config.Get(ctx, "test/types#i")
		if err != nil {
			t.Fatalf("did not expect an error: %s", err)
		}

		got, ok := v.Float64()
		if !ok {
			t.Fatalf("did not expect not being able to convert value to float 64")
		}

		expected := float64(9)
		if got != expected {
			t.Fatalf("expected '%f'; got '%f'", expected, got)
		}
	})

	t.Run("we can get a coerced float 64 value", func(t *testing.T) {
		v, err := config.Get(ctx, "test/types#ii")
		if err != nil {
			t.Fatalf("did not expect an error: %s", err)
		}

		got, ok := v.Float64()
		if !ok {
			t.Fatalf("did not expect not being able to coerce value to an float 64")
		}

		expected := float64(8)
		if got != expected {
			t.Fatalf("expected '%f'; got '%f'", expected, got)
		}
	})

	t.Run("we cannot get a coerced float 64 value", func(t *testing.T) {
		v, err := config.Get(ctx, "test/types#iii")
		if err != nil {
			t.Fatalf("did not expect an error: %s", err)
		}

		got, ok := v.Float64()
		if ok {
			t.Fatalf("did not expect being able to coerce value to an float 64 from a random string")
		}

		expected := float64(0)
		if got != expected {
			t.Fatalf("expected '%f'; got '%f'", expected, got)
		}
	})

	t.Run("we can get a string map", func(t *testing.T) {
		v, err := config.Get(ctx, "test/types#m")
		if err != nil {
			t.Fatalf("did not expect an error: %s", err)
		}

		got, ok := v.Map()
		if !ok {
			t.Fatalf("did not expect not being able to convert value to a string map; %T", v.Raw())
		}

		expected := "tres"
		if got["three"] != expected {
			t.Fatalf("expected '%s'; got '%s'", expected, got["three"])
		}
	})

	t.Run("we cannot get a string map", func(t *testing.T) {
		v, err := config.Get(ctx, "test/types#mm")
		if err != nil {
			t.Fatalf("did not expect an error: %s", err)
		}

		_, ok := v.Map()
		if ok {
			t.Fatalf("did not expect being able to convert incompatible value to a string map")
		}
	})

	t.Run("we can get a string slice", func(t *testing.T) {
		v, err := config.Get(ctx, "test/types#l")
		if err != nil {
			t.Fatalf("did not expect an error: %s", err)
		}

		got, ok := v.StringSlice()
		if !ok {
			t.Fatalf("did not expect not being able to convert value to a string slice; %T", v.Raw())
		}

		if len(got) != 3 {
			t.Fatalf("length of string slice should be 3")
		}

		expected := "three"
		if got[2] != expected {
			t.Fatalf("expected '%s'; got '%s'", expected, got[2])
		}
	})
	t.Run("we cannot get a string slice", func(t *testing.T) {
		v, err := config.Get(ctx, "test/types#ll")
		if err != nil {
			t.Fatalf("did not expect an error: %s", err)
		}

		_, ok := v.StringSlice()
		if ok {
			t.Fatalf("did not expect being able to convert incompatible value to a string slice")
		}
	})

	t.Run("we can get a duration value", func(t *testing.T) {
		v, err := config.Get(ctx, "test/types#d")
		if err != nil {
			t.Fatalf("did not expect an error: %s", err)
		}

		got, ok := v.Duration()
		if !ok {
			t.Fatalf("did not expect not being able to convert value to a time duration")
		}

		expected := 7 * time.Second
		if got != expected {
			t.Fatalf("expected '%s'; got '%s'", expected, got)
		}
	})

	t.Run("we cannot get a duration value", func(t *testing.T) {
		v, err := config.Get(ctx, "test/types#dd")
		if err != nil {
			t.Fatalf("did not expect an error: %s", err)
		}

		got, ok := v.Duration()
		if ok {
			t.Fatalf("did not expect being able to convert incompatible value to a time duration")
		}

		expected := time.Duration(0)
		if got != expected {
			t.Fatalf("expected '%s'; got '%s'", expected, got)
		}
	})

	t.Run("we can get the expected integer from the raw value", func(t *testing.T) {
		v, err := config.Get(ctx, "test/types#i")
		if err != nil {
			t.Fatalf("did not expect an error: %s", err)
		}

		raw := v.Raw()
		if raw == nil {
			t.Fatalf("did not expect a nil raw value")
		}

		n, ok := raw.(json.Number)
		if !ok {
			t.Fatalf("did not get expected type from the raw value; %T", raw)
		}

		i, err := n.Int64()
		if err != nil {
			t.Fatalf("could not extract int64 from json number: %s", err)
		}

		if i != 9 {
			t.Fatalf("raw value did not turn out to be the expeced underlying value")
		}
	})

	t.Run("we can get the expected boolean from the raw value", func(t *testing.T) {
		v, err := config.Get(ctx, "test/types#b")
		if err != nil {
			t.Fatalf("did not expect an error: %s", err)
		}

		raw := v.Raw()
		if raw == nil {
			t.Fatalf("did not expect a nil raw value")
		}

		b, ok := raw.(bool)
		if !ok {
			t.Fatalf("did not get expected type from the raw value; %T", raw)
		}

		if b != true {
			t.Fatalf("raw value did not turn out to be the expeced underlying value")
		}
	})

	t.Run("we can get a data map from the vault path", func(t *testing.T) {
		v, err := config.Get(ctx, "test/types")
		if err != nil {
			t.Fatalf("did not expect an error: %s", err)
		}

		m, ok := v.Data()
		if !ok {
			t.Fatalf("did not expect not being able to get a data map")
		}

		if len(m) == 0 {
			t.Fatalf("got an empty data map")
		}

		s, ok := m["bbb"]
		if !ok {
			t.Fatalf("could not get bbb key from the map")
		}

		if fmt.Sprintf("%s", s) != "abracadabra" {
			t.Fatalf("did not get expected value from the bbb key")
		}
	})

	t.Run("we cannot get a data map if the field is not a data map", func(t *testing.T) {
		v, err := config.Get(ctx, "test/types#ll")
		if err != nil {
			t.Fatalf("did not expect an error: %s", err)
		}

		m, ok := v.Data()
		if ok {
			t.Fatalf("did not expect to be able to get a data map")
		}

		if len(m) != 0 {
			t.Fatalf("should get an empty data map")
		}
	})
}

func TestConfyWithOverride(t *testing.T) {
	config := New(NewVaultClient(), 2*time.Minute, true)
	defer config.Close()
	ctx := context.Background()

	t.Run("expect value override from environment", func(t *testing.T) {
		other := "some-other-user"
		t.Setenv("TEST_APP_USER", other)
		v, err := config.Get(ctx, "test/app#user")
		if err != nil {
			t.Fatalf("%s", err)
		}

		if v.String() != other {
			t.Fatalf("on test/app#user; expected '%s'; got '%s'", other, v.String())
		}
	})
}

func TestConfyWatch(t *testing.T) {
	client := NewVaultClient()
	config := new(client, 1*time.Second, false)
	defer config.Close()
	signal := make(chan struct{}, 1)

	val, err := config.Get(context.Background(), "test/app#password")
	if err != nil {
		t.Fatalf("did not expect an error here")
	}

	t.Logf("existing value is '%s'", val.String())
	defer func() {
		// restore values
		err := client.RawClient().KVv1("secret").Put(context.Background(), "test/app", map[string]any{
			"user":     "fake-user",
			"password": val.String(),
		})
		if err != nil {
			t.Logf("could not restore values: %s", err)
		}
	}()

	cancel := config.Watch("test/app#password", func(oldVal, newVal Value) bool {
		return oldVal.String() != newVal.String()
	}, func(v Value) {
		t.Logf("value changed to '%s'", v.String())
		signal <- struct{}{}
	})
	defer cancel()

	go func() {
		err := client.RawClient().KVv1("secret").Put(context.Background(), "test/app", map[string]any{
			"user":     "fake-user",
			"password": "password is changed",
		})
		if err != nil {
			t.Logf("could not change values: %s", err)
		}
	}()

	select {
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for watcher to catch a change")
	case <-signal:
	}
}

func TestConfyClose(t *testing.T) {
	config := New(NewVaultClient(), 2*time.Minute, false)
	c := config.(*confyImpl)
	if c.closed {
		t.Fatalf("expected it to not be closed")
	}
	config.Close()
	if !c.closed {
		t.Fatalf("expected it to be closed")
	}
}
