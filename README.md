sensord
=========

A simple HTTP monitoring tool. Very much a WIP.

## Running locally

(tested on debian 7)

```
$ go get github.com/gorsuch/sensord
...
$ export LOCATION=my_house
$ export CHECKS_URL=https://s3.amazonaws.com/canary-public-data/data.json
$ go run sensord.go
```
