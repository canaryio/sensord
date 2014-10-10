package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/canaryio/sensord/pkg/manifest"
	"github.com/canaryio/sensord/pkg/registry"
	"github.com/canaryio/sensord/pkg/sampler"
	"github.com/kr/s3"
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

func buffer(d time.Duration, ingress chan *sampler.Sample, egress chan []*sampler.Sample) {
	t := time.NewTicker(d)
	samples := make([]*sampler.Sample, 0)

	for {
		select {
		case s := <-ingress:
			samples = append(samples, s)
		case <-t.C:
			egress <- samples
			samples = make([]*sampler.Sample, 0)
		}
	}
}

func s3Sink(ingress chan []*sampler.Sample) {
	s3Keys := s3.Keys{
		AccessKey: os.Getenv("AWS_ACCESS_KEY"),
		SecretKey: os.Getenv("AWS_SECRET_KEY"),
	}

	s3BaseURL, err := url.Parse("https://canary-buffers-us-east-1.s3.amazonaws.com")
	if err != nil {
		log.Fatal(err)
	}

	for s := range ingress {
		k := time.Now().Format(time.RFC3339Nano)
		s3URL := s3BaseURL
		s3URL.Path = k

		b, err := json.Marshal(s)
		if err != nil {
			log.Printf("buffer error=%v", err)
			continue
		}

		data := bytes.NewBuffer(b)
		r, _ := http.NewRequest("PUT", s3URL.String(), data)
		r.ContentLength = int64(data.Len())
		r.Header.Set("Date", time.Now().UTC().Format(http.TimeFormat))
		s3.Sign(r, s3Keys)

		resp, err := http.DefaultClient.Do(r)
		if err != nil {
			log.Fatal(err)
		}
		resp.Body.Close()

		log.Printf("buffer at=put url=%s status_code=%d", s3URL.String(), resp.StatusCode)
	}
}

func main() {
	config := getConfig()
	manifestChan := make(chan *manifest.Manifest)

	// the registry houses all of our samplers
	reg := registry.New(config.location)

	// buffer things
	fromBuffer := make(chan []*sampler.Sample)
	go buffer(5*time.Second, reg.C, fromBuffer)

	// send on to S3
	go s3Sink(fromBuffer)

	// keep fetching the manifests
	go manifest.ContinuouslyGet(config.manifestURL, manifestChan)
	go func() {
		for m := range manifestChan {
			reg.Update(m)
		}
	}()

	select {}
}
