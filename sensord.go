package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
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

func location() string {
	l := os.Getenv("LOCATION")
	if len(l) == 0 {
		fmt.Fprintf(os.Stderr, "LOCATION not defined in ENV\n")
		os.Exit(1)
	}

	return l
}

func checks_url() string {
	l := os.Getenv("CHECKS_URL")
	if len(l) == 0 {
		fmt.Fprintf(os.Stderr, "CHECKS_URL not defined in ENV\n")
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

func measurer(checks chan check, measurements chan measurement) {
	for {
		c := <-checks
		m := measure(c)

		measurements <- m
	}
}

func recorder(measurements chan measurement) {
	var payload []measurement
	for {
		m := <-measurements
		payload = payload.append(payload, m)

		s, err := json.Marshal(&payload)
		if err != nil {
			panic(err)
		}

		body := bytes.NewBuffer(s)
		req, err := http.NewRequest("POST", "http://localhost:5000/measurements", body)
		if err != nil {
			panic(err)
		}

		req.Header.Add("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		resp.Body.Close()
		if err != nil {
			panic(err)
		}

		fmt.Println(resp)
	}
}

func get_checks() []check {
	url := checks_url()

	res, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}

	var checks []check
	err = json.Unmarshal(body, &checks)
	if err != nil {
		panic(err)
	}

	return checks
}

func main() {
	check_list := get_checks()

	checks := make(chan check)
	measurements := make(chan measurement)

	go measurer(checks, measurements)
	go recorder(measurements)

	for {
		for _, c := range check_list {
			checks <- c
		}

		time.Sleep(1000 * time.Millisecond)
	}
}
