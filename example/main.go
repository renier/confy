package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mitchellh/mapstructure"

	"github.com/renier/confy"
)

func main() {
	config := confy.New(confy.NewVaultClient(), 5 * time.Second, false)
	defer config.Close()

	// Setup a watch for the debug flag
	cancel := config.Watch("search/test/app#debug", func(oldval, newval confy.Value) bool {
		old, _ := oldval.Bool()
		cur, _ := newval.Bool()
		return old != cur
	}, func(v confy.Value) {
		log.Printf("Debug changed to '%t'\n", v.Raw())
	})
	defer cancel()

	// Show how you can get arbitrary data from vault into
	// a custom struct
	data, err := config.Get(context.Background(), "search/test/app")
	if err != nil {
		panic(err)
	}

	var appConfig AppConfig
	dec, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.RecursiveStructToMapHookFunc(),
		),
		Result: &appConfig,
	})
	if err != nil {
		panic(err)
	}

	m, ok := data.Data()
	if !ok {
		panic("could not get data map")
	}

	err = dec.Decode(m)
	if err != nil {
		panic(err)
	}
	log.Printf("App Configuration received:\n%v\n", data)
	log.Printf("App Configuration unmarshaled:\n%+v\n", appConfig)

	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)
	<-sig
}

type AppConfig struct {
	Debug            bool              `mapstructure:"debug"`
	Host             string            `mapstructure:"host"`
	Port             int               `mapstructure:"port"`
	PrestoQueryDelay time.Duration     `mapstructure:"presto_query_delay"`
	LogFields        map[string]string `mapstructure:"log_fields"`
	MetricTags       []string          `mapstructure:"metric_tags"`
}
