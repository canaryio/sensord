package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/andelf/go-curl"
	"github.com/nu7hatch/gouuid"
	"github.com/rcrowley/go-metrics"
	"github.com/rcrowley/go-metrics/librato"
	"github.com/vmihailenco/msgpack"
)

var config Config

type Config struct {
	Location           string
	Targets            []string
	ChecksURL          string
	MeasurerCount      int
	MaxMeasurers       int
	LibratoEmail       string
	LibratoToken       string
	ToMeasurerTimer    metrics.Timer
	ToPusherTimer      metrics.Timer
	MeasurementCounter metrics.Counter
	PushCounter        metrics.Counter
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

func measurer(config Config, toMeasurer chan Check, toPusher chan Measurement) {
	for c := range toMeasurer {
		m := c.Measure(config)
		config.MeasurementCounter.Inc(1)
		config.ToPusherTimer.Time(func() { toPusher <- m })
	}
}

// listens for measurements on c, pushes them over UDP to addr
func udpPusher(addr string, c chan Measurement) {
	serverAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		panic(err)
	}

	con, err := net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("fn=udpPusher endpoint=%s\n", addr)

	for m := range c {
		payload, err := msgpack.Marshal(m)
		if err != nil {
			log.Fatal(err)
		}
		con.Write(payload)
	}
}

// listens for measurements on toPusher, fans them out to 1 or more addrs
func pusher(addrs []string, toPusher chan Measurement) {
	var chans []chan Measurement
	for _, addr := range addrs {
		c := make(chan Measurement)
		chans = append(chans, c)
		go udpPusher(addr, c)
	}

	for m := range toPusher {
		for _, c := range chans {
			c <- m
		}
		config.PushCounter.Inc(1)
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

func scheduler(config Config, check Check, toMeasurer chan Check) {
	for {
		config.ToMeasurerTimer.Time(func() { toMeasurer <- check })
		time.Sleep(1000 * time.Millisecond)
	}
}

func getEnvWithDefault(name, def string) string {
	val := os.Getenv(name)
	if val != "" {
		return val
	}

	return def
}

func init() {
	config.Location = getEnvWithDefault("LOCATION", "undefined")
	config.ChecksURL = getEnvWithDefault("CHECKS_URL", "https://s3.amazonaws.com/canary-public-data/checks.json")

	measurer_count, err := strconv.Atoi(getEnvWithDefault("MEASURER_COUNT", "1"))
	if err != nil {
		log.Fatal(err)
	}
	config.MeasurerCount = measurer_count

	config.Targets = strings.Split(os.Getenv("TARGETS"), ",")

	config.LibratoEmail = os.Getenv("LIBRATO_EMAIL")
	config.LibratoToken = os.Getenv("LIBRATO_TOKEN")

	config.ToMeasurerTimer = metrics.NewTimer()
	metrics.Register("sensord.to_measurer", config.ToMeasurerTimer)

	config.ToPusherTimer = metrics.NewTimer()
	metrics.Register("sensord.to_pusher", config.ToPusherTimer)

	config.MeasurementCounter = metrics.NewCounter()
	metrics.Register("sensord.measurements", config.MeasurementCounter)

	config.PushCounter = metrics.NewCounter()
	metrics.Register("sensord.push", config.PushCounter)
}

func main() {
	if config.LibratoEmail != "" && config.LibratoToken != "" {
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
	toPusher := make(chan Measurement)

	// spawn one scheduler per check
	for _, c := range checkList {
		go scheduler(config, c, toMeasurer)
	}

	// spawn N measurers
	for i := 0; i < config.MeasurerCount; i++ {
		go measurer(config, toMeasurer, toPusher)
	}

	// emit measurements to targets via UDP
	go pusher(config.Targets, toPusher)

	select {}
}
