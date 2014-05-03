package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/andelf/go-curl"
	"github.com/nu7hatch/gouuid"
)

type Config struct {
	Location         string
	ChecksUrl        string
	MeasurementsUrl  string
	MeasurementsUser string
	MeasurementsPass string
	MeasurerCount    int
	RecorderCount    int
}

type Check struct {
	Id  string `json:"id"`
	Url string `json:"url"`
}

type Measurement struct {
	Check             Check   `json:"check"`
	Id                string  `json:"id"`
	Location          string  `json:"location"`
	T                 int     `json:"t"`
	ExitStatus        int     `json:"exit_status"`
	ConnectTime       float64 `json:"connect_time,omitempty"`
	StartTransferTime float64 `json:"starttransfer_time,omitempty"`
	LocalIp           string  `json:"local_ip,omitempty"`
	PrimaryIp         string  `json:"primary_ip,omitempty"`
	TotalTime         float64 `json:"total_time,omitempty"`
	HttpStatus        int     `json:"http_status,omitempty"`
	NameLookupTime    float64 `json:"namelookup_time,omitempty"`
}

func (c *Check) Measure(config Config) Measurement {
	var m Measurement

	id, _ := uuid.NewV4()
	m.Id = id.String()
	m.Check = *c
	m.Location = config.Location

	easy := curl.EasyInit()
	defer easy.Cleanup()

	easy.Setopt(curl.OPT_URL, c.Url)

	// dummy func for curl output
	noOut := func(buf []byte, userdata interface{}) bool {
		return true
	}

	easy.Setopt(curl.OPT_WRITEFUNCTION, noOut)
	easy.Setopt(curl.OPT_CONNECTTIMEOUT, 10)
	easy.Setopt(curl.OPT_TIMEOUT, 10)

	now := time.Now()
	m.T = int(now.Unix())

	if err := easy.Perform(); err != nil {
		if e, ok := err.(curl.CurlError); ok {
			m.ExitStatus = (int(e))
			return m
		}
		os.Exit(1)
	}

	m.ExitStatus = 0
	http_status, _ := easy.Getinfo(curl.INFO_RESPONSE_CODE)
	m.HttpStatus = http_status.(int)

	connect_time, _ := easy.Getinfo(curl.INFO_CONNECT_TIME)
	m.ConnectTime = connect_time.(float64)

	namelookup_time, _ := easy.Getinfo(curl.INFO_NAMELOOKUP_TIME)
	m.NameLookupTime = namelookup_time.(float64)

	starttransfer_time, _ := easy.Getinfo(curl.INFO_STARTTRANSFER_TIME)
	m.StartTransferTime = starttransfer_time.(float64)

	total_time, _ := easy.Getinfo(curl.INFO_TOTAL_TIME)
	m.TotalTime = total_time.(float64)

	local_ip, _ := easy.Getinfo(curl.INFO_LOCAL_IP)
	m.LocalIp = local_ip.(string)

	primary_ip, _ := easy.Getinfo(curl.INFO_PRIMARY_IP)
	m.PrimaryIp = primary_ip.(string)

	return m
}

func measurer(config Config, toMeasurer chan Check, toRecorder chan Measurement) {
	for {
		c := <-toMeasurer
		m := c.Measure(config)

		toRecorder <- m
	}
}

func record(config Config, payload []Measurement) {
	s, err := json.Marshal(&payload)
	if err != nil {
		panic(err)
	}

	body := bytes.NewBuffer(s)
	req, err := http.NewRequest("POST", config.MeasurementsUrl, body)
	if err != nil {
		panic(err)
	}

	req.Header.Add("Content-Type", "application/json")

	if config.MeasurementsUser != "" {
		req.SetBasicAuth(config.MeasurementsUser, config.MeasurementsPass)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}

	fmt.Printf("fn=Record http_code=%d\n", resp.StatusCode)
	resp.Body.Close()
}

func recorder(config Config, toRecorder chan Measurement) {
	tickChan := time.NewTicker(time.Millisecond * 1000).C
	payload := make([]Measurement, 0, 100)

	for {
		select {
		case m := <-toRecorder:
			payload = append(payload, m)
		case <-tickChan:
			l := len(payload)
			fmt.Printf("fn=RecordLoop payload_size=%d\n", l)

			if l > 0 {
				record(config, payload)
				payload = make([]Measurement, 0, 100)
			}
		}
	}
}

func getChecks(config Config) []Check {
	url := config.ChecksUrl

	res, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}

	var checks []Check
	err = json.Unmarshal(body, &checks)
	if err != nil {
		panic(err)
	}

	return checks
}

func scheduler(check Check, toMeasurer chan Check) {
	for {
		toMeasurer <- check
		time.Sleep(1000 * time.Millisecond)
	}
}

func main() {
	config := Config{}
	flag.StringVar(&config.Location, "location", "undefined", "location of this sensor")
	flag.StringVar(&config.ChecksUrl, "checks_url", "https://s3.amazonaws.com/canary-public-data/checks.json", "URL for check data")
	flag.StringVar(&config.MeasurementsUrl, "measurements_url", "http://localhost:5000/measurements", "URL to POST measurements to")
	flag.IntVar(&config.MeasurerCount, "measurer_count", 1, "number of measurers to run")
	flag.IntVar(&config.RecorderCount, "recorder_count", 1, "number of recorders to run")
	flag.Parse()

	u, err := url.Parse(config.MeasurementsUrl)
	if err != nil {
		panic(err)
	}

	if u.User != nil {
		config.MeasurementsUser = u.User.Username()
		config.MeasurementsPass, _ = u.User.Password()
	}

	check_list := getChecks(config)

	toMeasurer := make(chan Check)
	toRecorder := make(chan Measurement)

	for i := 0; i < config.MeasurerCount; i++ {
		go measurer(config, toMeasurer, toRecorder)
	}

	for i := 0; i < config.RecorderCount; i++ {
		go recorder(config, toRecorder)
	}

	for _, c := range check_list {
		go scheduler(c, toMeasurer)
	}

	select {}
}
