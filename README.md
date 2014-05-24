sensord
=========

`canaryd` is a simple HTTP monitoring tool that watches a configurable set of URLs and emits measurement data via UDP to a configurable set of targets.

It is the basic building block of [canary.io](http://canary.io).

## Build

```sh
$ go get github.com/canaryio/sensord
$ cd $GOPATH/src/github.com/canaryio/sensord
$ godep get
$ godep go build
```

## Configuration

Check the local help for options:

```sh
$ ./sensord -h
Usage of ./sensord:
  -checks_url="https://s3.amazonaws.com/canary-public-data/checks.json": URL for check data
  -location="undefined": location of this sensor
  -measurer_count=1: number of measurers to run
```

Measurement destinations can be configured via the environment:

```
export TARGETS=host1.example.com:5000,host2.example.com:1310
```

`sensord` allows operational metrics to be sent to Librato.  You can configure with the following environment variables:

```
export LIBRATO_EMAIL=me@mydomain.com
export LIBRATO_TOKEN=asdf
export LIBRATO_SOURCE=my_hostname
```

## Usage Example

```sh
$ TARGETS=localhost:5000 godep go run sensord.go
2014/05/24 13:58:03 fn=udpPusher endpoint=localhost:5000
```
