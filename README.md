sensord
=========

A simple HTTP monitoring tool for [canary.io](http://canary.io).

## Running locally

```sh
$ go get github.com/gorsuch/sensord
$ go run sensord.go
```

## Configuration

Check the local help for options:

```sh
$ ./sensord -h
Usage of ./sensord:
  -checks_url="https://s3.amazonaws.com/canary-public-data/checks.json": URL for check data
  -location="undefined": location of this sensor
  -measurements_url="http://localhost:5000/measurements": URL to POST measurements to
  -measurer_count=1: number of measurers to run
  -recorder_count=1: number of recorders to run
```
