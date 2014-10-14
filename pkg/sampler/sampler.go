package sampler

import (
	"time"

	"github.com/andelf/go-curl"
)

type Sample struct {
	Name              string
	URL               string
	Source            string
	T                 int64
	ExitStatus        int
	HTTPStatus        int
	TotalTime         float64
	NameLookupTime    float64
	ConnectTime       float64
	StartTransferTime float64
	IP                string
	SizeDownload      float64
}

type Sampler struct {
	easy *curl.CURL
}

func New() *Sampler {
	return &Sampler{easy: curl.EasyInit()}
}

func (s *Sampler) Sample(name, url, location string) (*Sample, error) {
	defer s.easy.Reset()

	m := &Sample{}
	m.Name = name
	m.Source = location
	m.URL = url
	m.T = time.Now().Unix()

	s.easy.Setopt(curl.OPT_URL, url)

	// dummy func for curl output
	noOut := func(buf []byte, userdata interface{}) bool {
		return true
	}

	s.easy.Setopt(curl.OPT_WRITEFUNCTION, noOut)
	s.easy.Setopt(curl.OPT_CONNECTTIMEOUT, 10)
	s.easy.Setopt(curl.OPT_TIMEOUT, 10)

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
	m.ConnectTime = connectTime.(float64)

	namelookupTime, _ := s.easy.Getinfo(curl.INFO_NAMELOOKUP_TIME)
	m.NameLookupTime = namelookupTime.(float64)

	starttransferTime, _ := s.easy.Getinfo(curl.INFO_STARTTRANSFER_TIME)
	m.StartTransferTime = starttransferTime.(float64)

	totalTime, _ := s.easy.Getinfo(curl.INFO_TOTAL_TIME)
	m.TotalTime = totalTime.(float64)

	primaryIP, _ := s.easy.Getinfo(curl.INFO_PRIMARY_IP)
	m.IP = primaryIP.(string)

	sizeDownload, _ := s.easy.Getinfo(curl.INFO_SIZE_DOWNLOAD)
	m.SizeDownload = sizeDownload.(float64)

	return m, nil
}
