package router

import (
	"fmt"
	"sync"

	"github.com/canaryio/sensord/pkg/logfmtservice"
	"github.com/canaryio/sensord/pkg/manifest"
	"github.com/canaryio/sensord/pkg/sampler"
	"github.com/canaryio/sensord/pkg/service"
)

type Router struct {
	sync.Mutex
	sites    map[string]*manifest.SiteDefinition
	services map[string]service.Service
	manifest *manifest.Manifest
}

func New() *Router {
	return &Router{
		sites:    make(map[string]*manifest.SiteDefinition),
		services: make(map[string]service.Service),
	}
}

func (r *Router) Ingest(s *sampler.Sample) error {
	r.Lock()
	defer r.Unlock()

	site := r.sites[s.Site.ID]

	if site == nil {
		return fmt.Errorf("unknown site_id: %s", s.Site.ID)
	}

	for _, svcID := range site.Services {
		svc := r.services[svcID]
		if svc == nil {
			return fmt.Errorf("unknown service_id: %s", svcID)
		}

		svc.Ingest(s)
	}

	return nil
}

func (r *Router) Update(m *manifest.Manifest) {
	r.Lock()
	defer r.Unlock()

	for _, s := range m.Sites {
		if r.sites[s.ID] == nil {
			r.sites[s.ID] = s
		}
	}

	for _, s := range m.Services {
		if r.services[s.ID] == nil {
			svc := logfmtservice.New()
			svc.Start()
			r.services[s.ID] = svc
		}
	}
}
