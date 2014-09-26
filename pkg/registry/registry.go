package registry

import (
	"sync"

	"github.com/canaryio/sensord/pkg/agent"
	"github.com/canaryio/sensord/pkg/manifest"
	"github.com/canaryio/sensord/pkg/sampler"
)

type Registry struct {
	sync.Mutex
	location string
	outChan  chan *sampler.Sample
	agents   map[string]*agent.Agent
}

func New(location string, outChan chan *sampler.Sample) *Registry {
	return &Registry{
		location: location,
		outChan:  outChan,
		agents:   make(map[string]*agent.Agent),
	}
}

func (r *Registry) add(site *manifest.SiteDefinition) {
	r.Lock()
	defer r.Unlock()

	if r.agents[site.ID] == nil {
		agent := agent.New(r.location, site, r.outChan)
		r.agents[site.ID] = agent
		agent.Start()
	}
}

func (r *Registry) remove(siteID string) {
	r.Lock()
	defer r.Unlock()

	a := r.agents[siteID]
	if a != nil {
		a.Stop()
		r.agents[siteID] = nil
	}
}

func (r *Registry) Update(m *manifest.Manifest) {
	for _, site := range m.Sites {
		r.add(site)
	}

	sm := m.SiteMap()
	// remove site if it is no longer in the Manifest
	for k, _ := range r.agents {
		if sm[k] == nil {
			r.remove(k)
		}
	}
}
