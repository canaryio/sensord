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

`sensord` is configured via the environment.  The following values are allowed:

* `CHECKS_URL` - location of checks data, defaults to https://s3.amazonaws.com/canary-public-data/checks.json
* `TARGETS` - comma separated list of host:port pairs to send measurements to
* `LOCATION` - name of this location, defaults to 'undefined'
* `MEASURER_COUNT` - number of measurers to run. Defaults to '1'
* `CHECK_PERIOD` - delay between checks, in ms. Defaults to '1000' (1 second)

`sensord` allows operational metrics to be sent to Librato. You can enable this by configured the following environment variables:

* `LIBRATO_EMAIL` - email address of your librato account
* `LIBRATO_TOKEN` - token for your Librato account

If you'd like to log metrics to `STDERR`, you can do so by setting `LOGSTDERR` to '1'.

## Usage Example

```sh
$ TARGETS=localhost:5000 godep go run sensord.go
2014/05/24 13:58:03 fn=udpPusher endpoint=localhost:5000
```
