package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/bmizerany/aws4"
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
		l := data.Len()

		resp, err := s3Put(s3URL.String(), data)
		if err != nil {
			log.Fatal(err)
		}
		resp.Body.Close()

		log.Printf("s3sink count=%d size=%d at=put url=%s status_code=%d", len(s), l, s3URL.String(), resp.StatusCode)

		topic := "arn:aws:sns:us-east-1:854436987475:new-buffer"
		endpoint := "https://sns.us-east-1.amazonaws.com"
		resp, err = snsPublish(endpoint, topic, "buffer", s3URL.String())
		if err != nil {
			log.Fatal(err)
		}
		resp.Body.Close()

		log.Printf("s3sink at=publish endpoint=%s topic=%s status_code=%d", endpoint, topic, resp.StatusCode)
	}
}

func s3Put(s3URL string, buf *bytes.Buffer) (resp *http.Response, err error) {
	s3Keys := s3.Keys{
		AccessKey: os.Getenv("AWS_ACCESS_KEY"),
		SecretKey: os.Getenv("AWS_SECRET_KEY"),
	}

	r, _ := http.NewRequest("PUT", s3URL, buf)
	r.ContentLength = int64(buf.Len())
	r.Header.Set("Date", time.Now().UTC().Format(http.TimeFormat))
	s3.Sign(r, s3Keys)

	return http.DefaultClient.Do(r)
}

func snsPublish(endpoint, topic, subject, message string) (resp *http.Response, err error) {
	v := url.Values{}
	v.Set("Action", "Publish")
	v.Set("TopicArn", topic)
	v.Set("Subject", subject)
	v.Set("Message", message)

	return aws4.PostForm(endpoint, v)
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
