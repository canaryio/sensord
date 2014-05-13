package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/andelf/go-curl"
	"github.com/nu7hatch/gouuid"
)

var config Config

type Config struct {
	Port          string
	Location      string
	ChecksUrl     string
	MeasurerCount int
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
	SizeDownload      float64 `json:"size_download,omitempty"`
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

	size_download, _ := easy.Getinfo(curl.INFO_SIZE_DOWNLOAD)
	m.SizeDownload = size_download.(float64)

	return m
}

func measurer(config Config, toMeasurer chan Check, toStreamer chan Measurement) {
	for {
		c := <-toMeasurer
		m := c.Measure(config)

		toStreamer <- m
	}
}

func streamer(config Config, toStreamer chan Measurement) {
	h := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
		for {
			err := enc.Encode(<-toStreamer)
			if err != nil {
				return
			}

			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	}

	http.HandleFunc("/measurements", h)

	log.Printf("fn=streamer listening=true port=%s\n", config.Port)
	err := http.ListenAndServe(":"+config.Port, nil)
	if err != nil {
		log.Fatal(err)
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

func init() {
	flag.StringVar(&config.Port, "port", "5000", "port the HTTP server should listen on")
	flag.StringVar(&config.Location, "location", "undefined", "location of this sensor")
	flag.StringVar(&config.ChecksUrl, "checks_url", "https://s3.amazonaws.com/canary-public-data/checks.json", "URL for check data")
	flag.IntVar(&config.MeasurerCount, "measurer_count", 1, "number of measurers to run")
}

func main() {
	flag.Parse()

	check_list := getChecks(config)

	toMeasurer := make(chan Check)
	toStreamer := make(chan Measurement)

	// spawn one scheduler per check
	for _, c := range check_list {
		go scheduler(c, toMeasurer)
	}

	// spawn N measurers
	for i := 0; i < config.MeasurerCount; i++ {
		go measurer(config, toMeasurer, toStreamer)
	}

	// stream measurements to clients over HTTP
	go streamer(config, toStreamer)

	select {}
}
