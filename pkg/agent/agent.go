package agent

import (
	"log"
	"time"

	"github.com/canaryio/sensord/pkg/manifest"
	"github.com/canaryio/sensord/pkg/sampler"
)

type Agent struct {
	location string
	site     *manifest.SiteDefinition
	sampler  *sampler.Sampler
	outChan  chan *sampler.Sample
	stopChan chan bool
}

func New(location string, site *manifest.SiteDefinition, outChan chan *sampler.Sample) *Agent {
	return &Agent{
		location: location,
		site:     site,
		sampler:  sampler.New(),
		outChan:  outChan,
		stopChan: make(chan bool),
	}
}

func (a *Agent) run() {
	sampleTicker := time.NewTicker(time.Second)
	for {
		select {
		case <-sampleTicker.C:
			log.Printf("agent site_id=%s at=sample", a.site.ID)
			s, err := a.sampler.Sample(a.site, a.location)
			if err != nil {
				log.Fatal(err)
			}
			a.outChan <- s
		case <-a.stopChan:
			log.Printf("agent site_id=%s at=stop", a.site.ID)
			return
		}
	}
}

func (a *Agent) Start() {
	go a.run()
}

func (a *Agent) Stop() {
	a.stopChan <- true
}
