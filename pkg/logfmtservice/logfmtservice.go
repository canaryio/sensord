package logfmtservice

import (
	"fmt"
	"time"

	"github.com/canaryio/sensord/pkg/sampler"
)

type Service struct{}

func New() *Service {
	return &Service{}
}

func (s *Service) Start() {
}

func (s *Service) Stop() {
}

func (s *Service) Ingest(sample *sampler.Sample) {
	fmt.Printf(
		"sample=true site_id=%s location=%s t=%s exit_status=%d http_status=%d tt=%f ttnl=%f ttc=%f ttfb=%f ip=%s size=%d\n",
		sample.Site.ID,
		sample.Location,
		sample.T.Format(time.RFC3339),
		sample.ExitStatus,
		sample.HTTPStatus,
		sample.TT,
		sample.TTNL,
		sample.TTC,
		sample.TTFB,
		sample.IP,
		int64(sample.SizeDownload),
	)
}
