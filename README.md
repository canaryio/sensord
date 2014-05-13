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
  -http_basic_password="": HTTP basic authentication password
  -http_basic_realm="": HTTP basic authentication realm
  -http_basic_username="": HTTP basic authentication username
  -port="5000": port the HTTP server should listen on
  -location="undefined": location of this sensor
  -measurer_count=1: number of measurers to run
```
