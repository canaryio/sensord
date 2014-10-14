package pool

import (
	"sync"

	"github.com/canaryio/sensord/pkg/sampler"
	"github.com/canaryio/sensord/pkg/sensor"
)

type Registry struct {
	sync.Mutex
	source  string
	C       chan *sampler.Sample
	sensors map[string]*sensor.Sensor
}

func New(source string) *Registry {
	return &Registry{
		C:       make(chan *sampler.Sample),
		source:  source,
		sensors: make(map[string]*sensor.Sensor),
	}
}

func (r *Registry) add(name, url string) {
	r.Lock()
	defer r.Unlock()

	if r.sensors[name] == nil {
		sensor := sensor.New(r.source, name, url, r.C)
		r.sensors[name] = sensor
		sensor.Start()
	}
}

func (r *Registry) remove(siteID string) {
	r.Lock()
	defer r.Unlock()

	a := r.sensors[siteID]
	if a != nil {
		a.Stop()
		r.sensors[siteID] = nil
	}
}

func (r *Registry) Update(m map[string]*string) {
	for k, v := range m {
		r.add(k, *v)
	}

	// remove site if it is no longer in the Manifest
	for k, _ := range r.sensors {
		if m[k] == nil {
			r.remove(k)
		}
	}
}
