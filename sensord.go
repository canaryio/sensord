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

	"github.com/rcrowley/go-metrics"
	"github.com/rcrowley/go-metrics/influxdb"
	"github.com/rcrowley/go-metrics/librato"
	"github.com/vmihailenco/msgpack"
	"gopkg.in/canaryio/data.v2"
	"gopkg.in/canaryio/measure.v3/curl"
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
	InfluxdbHost       string
	InfluxdbDatabase   string
	InfluxdbUser       string
	InfluxdbPassword   string
	LogStderr          bool
	ToMeasurerTimer    metrics.Timer
	ToPusherTimer      metrics.Timer
	MeasurementCounter metrics.Counter
	PushCounter        metrics.Counter
	CheckPeriod        time.Duration
}

// listens for measurements on c, pushes them over UDP to addr
func udpPusher(addr string, c chan *data.Measurement) {
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
func pusher(addrs []string, toPusher chan *data.Measurement) {
	var chans []chan *data.Measurement
	for _, addr := range addrs {
		c := make(chan *data.Measurement)
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

func getChecks(config Config) []data.Check {
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

	var checks []data.Check
	err = json.Unmarshal(body, &checks)
	if err != nil {
		log.Fatal(err)
	}

	return checks
}

func scheduler(config Config, check *data.Check, measurer *curl.Measurer, toPusher chan *data.Measurement) {
	for {
		m, _ := measurer.Measure(check)
		config.ToMeasurerTimer.Time(func() { toPusher <- m })
		time.Sleep(config.CheckPeriod * time.Millisecond)
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

	check_period, err := strconv.Atoi(getEnvWithDefault("CHECK_PERIOD", "1000"))
	if err != nil {
		log.Fatal(err)
	}
	config.CheckPeriod = time.Duration(check_period)

	config.Targets = strings.Split(os.Getenv("TARGETS"), ",")

	config.LibratoEmail = os.Getenv("LIBRATO_EMAIL")
	config.LibratoToken = os.Getenv("LIBRATO_TOKEN")

	config.InfluxdbHost = os.Getenv("INFLUXDB_HOST")
	config.InfluxdbDatabase = os.Getenv("INFLUXDB_DATABASE")
	config.InfluxdbUser = os.Getenv("INFLUXDB_USER")
	config.InfluxdbPassword = os.Getenv("INFLUXDB_PASSWORD")

	if os.Getenv("LOGSTDERR") == "1" {
		config.LogStderr = true
	}

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
		log.Println("fn=main metrics=librato")
		go librato.Librato(metrics.DefaultRegistry,
			10e9,                  // interval
			config.LibratoEmail,   // account owner email address
			config.LibratoToken,   // Librato API token
			config.Location,       // source
			[]float64{50, 95, 99}, // precentiles to send
			time.Millisecond,      // time unit
		)
	}

	if config.InfluxdbHost != "" &&
		config.InfluxdbDatabase != "" &&
		config.InfluxdbUser != "" &&
		config.InfluxdbPassword != "" {
		log.Println("fn=main metrics=influxdb")

		go influxdb.Influxdb(metrics.DefaultRegistry, 10e9, &influxdb.Config{
			Host:     config.InfluxdbHost,
			Database: config.InfluxdbDatabase,
			Username: config.InfluxdbUser,
			Password: config.InfluxdbPassword,
		})
	}

	if config.LogStderr == true {
		go metrics.Log(metrics.DefaultRegistry, 10e9, log.New(os.Stderr, "metrics: ", log.Lmicroseconds))
	}

	checkList := getChecks(config)

	toPusher := make(chan *data.Measurement)

	measurer := curl.NewMeasurer(config.Location, config.MeasurerCount)

	// spawn one scheduler per check
	for _, c := range checkList {
		go scheduler(config, &c, measurer, toPusher)
	}

	// emit measurements to targets via UDP
	go pusher(config.Targets, toPusher)

	select {}
}
