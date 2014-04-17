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

func measure(c check) measurement {
	var m measurement

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
			fmt.Printf("%v\n", int(e))
		}
		os.Exit(1)
	}

	id, _ := uuid.NewV4()
	m.Id = id.String()
	m.CheckId = c.Id

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
