package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"github.com/nu7hatch/gouuid"
)

type check struct {
	Id  string `json:"id"`
	Url string `json:"url"`
}

// {"url": "http://github.com", "connect_time": 0.031178999999999998, "exit_status": 0, "starttransfer_time": 0.031178999999999998, "t": 1397688864, "local_ip": "107.170.123.131", "primary_ip": "192.30.252.128", "total_time": 0.037648000000000001, "http_status": 301, "namelookup_time": 0.024646000000000001, "local_port": 53858}
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

func curly(url string) []byte {
	cmd := exec.Command("curly", url)
	out, err := cmd.Output()
	if err != nil {
		panic(err)
	}

	return out
}

func measure(c check) measurement {
	s := curly(c.Url)

	var m measurement
	if err := json.Unmarshal(s, &m); err != nil {
		panic(err)
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
