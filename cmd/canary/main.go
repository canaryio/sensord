package main

import (
	"fmt"
	"log"
	"os"

	"github.com/canaryio/sensord/pkg/manifest"
	"github.com/canaryio/sensord/pkg/pool"
)

type config struct {
	location    string
	manifestURL string
}

func getConfig() *config {
	config := &config{}
	config.location = os.Getenv("LOCATION")
	if config.location == "" {
		log.Fatal("LOCATION must be set in ENV")
	}

	config.manifestURL = os.Getenv("MANIFEST_URL")
	if config.manifestURL == "" {
		log.Fatal("MANIFEST_URL must be set in ENV")
	}

	return config
}

func main() {
	config := getConfig()
	manifestChan := make(chan map[string]*string)

	// the pool houses all of our samplers
	reg := pool.New(config.location)

	go func() {
		for m := range reg.C {
			fmt.Fprintf(os.Stdout, "measurement=true name=\"%s\" url=\"%s\" source=\"%s\" t=%d exit_status=%d http_status=%d total_time=%f namelookup_time=%f connect_time=%f starttransfer_time=%f ip=\"%s\"\n",
				m.Name,
				m.URL,
				m.Source,
				m.T,
				m.ExitStatus,
				m.HTTPStatus,
				m.TotalTime,
				m.NameLookupTime,
				m.ConnectTime,
				m.StartTransferTime,
				m.IP,
			)
		}
	}()

	go manifest.ContinuouslyGet(config.manifestURL, manifestChan)
	go func() {
		for m := range manifestChan {
			reg.Update(m)
		}
	}()

	select {}
}
