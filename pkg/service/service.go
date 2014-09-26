package service

import "github.com/canaryio/sensord/pkg/sampler"

type Service interface {
	Start()
	Stop()
	Ingest(s *sampler.Sample)
}
