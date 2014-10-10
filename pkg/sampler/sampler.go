package sampler

import (
	"time"

	"github.com/andelf/go-curl"
	"github.com/canaryio/sensord/pkg/manifest"
)

type Sample struct {
	SiteID       string
	Location     string
	T            time.Time
	ExitStatus   int
	HTTPStatus   int
	TT           float64
	TTC          float64
	TTNL         float64
	TTFB         float64
	IP           string
	SizeDownload float64
}

type Sampler struct {
	easy *curl.CURL
}

func New() *Sampler {
	return &Sampler{easy: curl.EasyInit()}
}

func (s *Sampler) Sample(site *manifest.SiteDefinition, location string) (*Sample, error) {
	defer s.easy.Reset()

	m := &Sample{}
	m.SiteID = site.ID
	m.Location = location
	m.T = time.Now()

	s.easy.Setopt(curl.OPT_URL, site.URL)

	// dummy func for curl output
	noOut := func(buf []byte, userdata interface{}) bool {
		return true
	}

	s.easy.Setopt(curl.OPT_WRITEFUNCTION, noOut)
	s.easy.Setopt(curl.OPT_CONNECTTIMEOUT, 2)
	s.easy.Setopt(curl.OPT_TIMEOUT, 2)

	if err := s.easy.Perform(); err != nil {
		if e, ok := err.(curl.CurlError); ok {
			m.ExitStatus = (int(e))
			return m, nil
		}
		return nil, err
	}

	httpStatus, _ := s.easy.Getinfo(curl.INFO_RESPONSE_CODE)
	m.HTTPStatus = httpStatus.(int)

	connectTime, _ := s.easy.Getinfo(curl.INFO_CONNECT_TIME)
	m.TTC = connectTime.(float64)

	namelookupTime, _ := s.easy.Getinfo(curl.INFO_NAMELOOKUP_TIME)
	m.TTNL = namelookupTime.(float64)

	starttransferTime, _ := s.easy.Getinfo(curl.INFO_STARTTRANSFER_TIME)
	m.TTFB = starttransferTime.(float64)

	totalTime, _ := s.easy.Getinfo(curl.INFO_TOTAL_TIME)
	m.TT = totalTime.(float64)

	primaryIP, _ := s.easy.Getinfo(curl.INFO_PRIMARY_IP)
	m.IP = primaryIP.(string)

	sizeDownload, _ := s.easy.Getinfo(curl.INFO_SIZE_DOWNLOAD)
	m.SizeDownload = sizeDownload.(float64)

	return m, nil
}
