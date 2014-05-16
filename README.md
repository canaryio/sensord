sensord
=========

`canaryd` is a simple HTTP monitoring tool that watches a configurable set of URLs and streams the results via a local webserver to any connected clients.

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
  -http_basic_password="": HTTP basic authentication password
  -http_basic_realm="": HTTP basic authentication realm
  -http_basic_username="": HTTP basic authentication username
  -port="5000": port the HTTP server should listen on
  -location="undefined": location of this sensor
  -measurer_count=1: number of measurers to run
```


`sensord` allows metrics to be recorded to Librato.  You can configure with the following environment variables:

```
export LIBRATO_EMAIL=me@mydomain.com
export LIBRATO_TOKEN=asdf
export LIBRATO_SOURCE=my_hostname
```

## Usage Example

```sh
# http basic auth is required to access the stream
$ htpasswd -bn user pass
user:$apr1$2mSJKjRD$LFvgR6LUV2O1MfnLUQXlq1

$ godep go run sensord.go --http_basic_username user --http_basic_password '$apr1$2mSJKjRD$LFvgR6LUV2O1MfnLUQXlq1' &
[1] 9230
2014/05/13 04:25:12 fn=streamer listening=true port=5000

# measurements are streamed down via chunked encoding
$ curl http://user:pass@localhost:5000/measurements
{"check":{"id":"http-github.com","url":"http://github.com"},"id":"1aa3f596-0f18-4d2d-4dd8-3dd4448db5df","location":"undefined","t":1399955112,"exit_status":0,"connect_time":0.027653,"starttransfer_time":0.034589,"local_ip":"107.170.77.99","primary_ip":"192.30.252.129","total_time":0.03462,"http_status":301,"namelookup_time":0.020706}
{"check":{"id":"https-github.com","url":"https://github.com"},"id":"6b8d27d0-71e7-4345-745d-d5721b41d6e3","location":"undefined","t":1399955130,"exit_status":0,"connect_time":0.027542,"starttransfer_time":0.061631,"local_ip":"107.170.77.99","primary_ip":"192.30.252.130","total_time":0.068541,"http_status":200,"namelookup_time":0.020702,"size_download":15390}
^C
```
