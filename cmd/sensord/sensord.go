package main

import (
	"log"
	"os"

	"github.com/canaryio/sensord/pkg/manifest"
	"github.com/canaryio/sensord/pkg/registry"
	"github.com/canaryio/sensord/pkg/router"
	"github.com/canaryio/sensord/pkg/sampler"
)

type config struct {
	location    string
	manifestURL string
}

func getEnvWithDefault(name, def string) string {
	val := os.Getenv(name)
	if val != "" {
		return val
	}

	return def
}

func getConfig() *config {
	config := &config{}
	config.location = getEnvWithDefault("LOCATION", "undefined")
	config.manifestURL = os.Getenv("MANIFEST_URL")
	if config.manifestURL == "" {
		log.Fatal("MANIFEST_URL must be set in ENV")
	}

	return config
}

func main() {
	config := getConfig()
	manifestChan := make(chan *manifest.Manifest)
	sampleChan := make(chan *sampler.Sample)

	reg := registry.New(config.location, sampleChan)
	rt := router.New()

	// route our samples to the services
	go func() {
		for s := range sampleChan {
			rt.Ingest(s)
		}
	}()

	// keep fetching regularly fetch the manifests
	go manifest.ContinuouslyGet(config.manifestURL, manifestChan)
	go func() {
		for m := range manifestChan {
			rt.Update(m)
			reg.Update(m)
		}
	}()

	select {}
}
