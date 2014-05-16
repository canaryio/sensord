package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/abbot/go-http-auth"
	"github.com/andelf/go-curl"
	"github.com/nu7hatch/gouuid"
	"github.com/rcrowley/go-metrics"
	"github.com/rcrowley/go-metrics/librato"
)

var config Config

type Config struct {
	HTTPBasicUsername string
	HTTPBasicPassword string
	HTTPBasicRealm    string
	Port              string
	Location          string
	ChecksURL         string
	MeasurerCount     int
	LibratoEmail      string
	LibratoToken      string
	LibratoSource     string
}

type Check struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

type Measurement struct {
	Check             Check   `json:"check"`
	ID                string  `json:"id"`
	Location          string  `json:"location"`
	T                 int     `json:"t"`
	ExitStatus        int     `json:"exit_status"`
	ConnectTime       float64 `json:"connect_time,omitempty"`
	StartTransferTime float64 `json:"starttransfer_time,omitempty"`
	LocalIP           string  `json:"local_ip,omitempty"`
	PrimaryIP         string  `json:"primary_ip,omitempty"`
	TotalTime         float64 `json:"total_time,omitempty"`
	HTTPStatus        int     `json:"http_status,omitempty"`
	NameLookupTime    float64 `json:"namelookup_time,omitempty"`
	SizeDownload      float64 `json:"size_download,omitempty"`
}

func (c *Check) Measure(config Config) Measurement {
	var m Measurement

	id, _ := uuid.NewV4()
	m.ID = id.String()
	m.Check = *c
	m.Location = config.Location

	easy := curl.EasyInit()
	defer easy.Cleanup()

	easy.Setopt(curl.OPT_URL, c.URL)

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
	httpStatus, _ := easy.Getinfo(curl.INFO_RESPONSE_CODE)
	m.HTTPStatus = httpStatus.(int)

	connectTime, _ := easy.Getinfo(curl.INFO_CONNECT_TIME)
	m.ConnectTime = connectTime.(float64)

	namelookupTime, _ := easy.Getinfo(curl.INFO_NAMELOOKUP_TIME)
	m.NameLookupTime = namelookupTime.(float64)

	starttransferTime, _ := easy.Getinfo(curl.INFO_STARTTRANSFER_TIME)
	m.StartTransferTime = starttransferTime.(float64)

	totalTime, _ := easy.Getinfo(curl.INFO_TOTAL_TIME)
	m.TotalTime = totalTime.(float64)

	localIP, _ := easy.Getinfo(curl.INFO_LOCAL_IP)
	m.LocalIP = localIP.(string)

	primaryIP, _ := easy.Getinfo(curl.INFO_PRIMARY_IP)
	m.PrimaryIP = primaryIP.(string)

	sizeDownload, _ := easy.Getinfo(curl.INFO_SIZE_DOWNLOAD)
	m.SizeDownload = sizeDownload.(float64)

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
	a := func(user, realm string) string {
		if user == config.HTTPBasicUsername {
			return config.HTTPBasicPassword
		}
		return ""
	}

	h := func(w http.ResponseWriter, r *auth.AuthenticatedRequest) {
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

	authenticator := auth.NewBasicAuthenticator(config.HTTPBasicRealm, a)

	http.HandleFunc("/measurements", authenticator.Wrap(h))

	log.Printf("fn=streamer listening=true port=%s\n", config.Port)
	err := http.ListenAndServe(":"+config.Port, nil)
	if err != nil {
		log.Fatal(err)
	}
}

func getChecks(config Config) []Check {
	url := config.ChecksURL

	res, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	var checks []Check
	err = json.Unmarshal(body, &checks)
	if err != nil {
		log.Fatal(err)
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
	flag.StringVar(&config.HTTPBasicUsername, "http_basic_username", "", "HTTP basic authentication username")
	flag.StringVar(&config.HTTPBasicPassword, "http_basic_password", "", "HTTP basic authentication password")
	flag.StringVar(&config.HTTPBasicRealm, "http_basic_realm", "", "HTTP basic authentication realm")
	flag.StringVar(&config.Port, "port", "5000", "port the HTTP server should listen on")
	flag.StringVar(&config.Location, "location", "undefined", "location of this sensor")
	flag.StringVar(&config.ChecksURL, "checks_url", "https://s3.amazonaws.com/canary-public-data/checks.json", "URL for check data")
	flag.IntVar(&config.MeasurerCount, "measurer_count", 1, "number of measurers to run")

	config.LibratoEmail = os.Getenv("LIBRATO_EMAIL")
	config.LibratoToken = os.Getenv("LIBRATO_TOKEN")
}

func main() {
	flag.Parse()

	if len(config.HTTPBasicUsername) == 0 && len(config.HTTPBasicPassword) == 0 {
		log.Fatal("fatal - HTTP basic auth not set correctly")
	}

	if config.LibratoEmail != "" && config.LibratoToken != "" && config.LibratoSource != "" {
		log.Println("fn=main metircs=librato")
		go librato.Librato(metrics.DefaultRegistry,
			10e9,                  // interval
			config.LibratoEmail,   // account owner email address
			config.LibratoToken,   // Librato API token
			config.Location,       // source
			[]float64{50, 95, 99}, // precentiles to send
			time.Millisecond,      // time unit
		)
	}

	checkList := getChecks(config)

	toMeasurer := make(chan Check)
	toStreamer := make(chan Measurement)

	// spawn one scheduler per check
	for _, c := range checkList {
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
