package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/andelf/go-curl"
	"github.com/nu7hatch/gouuid"
)

type check struct {
	Id  string `json:"id"`
	Url string `json:"url"`
}

type measurement struct {
	Id                string  `json:"id"`
	CheckId           string  `json:"check_id"`
	Location          string  `json:"location"`
	Url               string  `json:"url"`
	ConnectTime       float64 `json:"connect_time"`
	ExitStatus        int     `json:"exit_status"`
	StartTransferTime float64 `json:"starttransfer_time"`
	T                 int     `json:"t"`
	LocalIp           string  `json:"local_ip"`
	PrimaryIp         string  `json:"primary_ip"`
	TotalTime         float64 `json:"total_time"`
	HttpStatus        int     `json:"http_status"`
	NameLookupTime    float64 `json:"namelookup_time"`
}

func location() string {
	l := os.Getenv("LOCATION")
	if len(l) == 0 {
		fmt.Fprintf(os.Stderr, "LOCATION not defined in ENV\n")
		os.Exit(1)
	}

	return l
}

func measure(c check) measurement {
	var m measurement

	id, _ := uuid.NewV4()
	m.Id = id.String()
	m.CheckId = c.Id
	m.Location = location()

	easy := curl.EasyInit()
	defer easy.Cleanup()

	easy.Setopt(curl.OPT_URL, c.Url)

	m.Url = c.Url

	// dummy func for curl output
	noOut := func(buf []byte, userdata interface{}) bool {
		return true
	}

	easy.Setopt(curl.OPT_WRITEFUNCTION, noOut)

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

func scheduler(checks chan check) {
	for {
		var c check
		c.Id = "1"
		c.Url = "http://github.com"

		checks <- c

		time.Sleep(1000 * time.Millisecond)
	}
}

func measurer(checks chan check, measurements chan measurement) {
	for {
		c := <-checks
		m := measure(c)

		measurements <- m
	}
}

func recorder(measurements chan measurement) {
	for {
		m := <-measurements

		s, err := json.Marshal(m)
		if err != nil {
			panic(err)
		}

		fmt.Println(string(s))
	}
}

func main() {
	checks := make(chan check)
	measurements := make(chan measurement)

	go scheduler(checks)
	go measurer(checks, measurements)
	go recorder(measurements)

	for {
		fmt.Println("ping...")
		time.Sleep(1000 * time.Millisecond)
	}
}
