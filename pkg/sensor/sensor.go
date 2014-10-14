package sensor

import (
	"log"
	"time"

	"github.com/canaryio/sensord/pkg/sampler"
)

type Sensor struct {
	source   string
	name     string
	url      string
	sampler  *sampler.Sampler
	outChan  chan *sampler.Sample
	stopChan chan bool
}

func New(source, name, url string, outChan chan *sampler.Sample) *Sensor {
	return &Sensor{
		source:   source,
		name:     name,
		url:      url,
		sampler:  sampler.New(),
		outChan:  outChan,
		stopChan: make(chan bool),
	}
}

func (a *Sensor) run() {
	sampleTicker := time.NewTicker(time.Second)
	for {
		select {
		case <-sampleTicker.C:
			s, err := a.sampler.Sample(a.name, a.url, a.source)
			if err != nil {
				log.Fatal(err)
			}
			a.outChan <- s
		case <-a.stopChan:
			return
		}
	}
}

func (a *Sensor) Start() {
	go a.run()
}

func (a *Sensor) Stop() {
	a.stopChan <- true
}
