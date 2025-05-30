<p>
  <a href="http://www.gwos.com/" target="blank"><img src=".github/img/readme_image.png" alt="GWOS Logo"/></a>
</p>

[![License](https://img.shields.io/github/license/gwos/tcg)](LICENSE)
[![Build](https://app.travis-ci.com/gwos/tcg.svg?branch=master)](https://app.travis-ci.com/gwos/tcg.svg?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/gwos/tcg)](https://goreportcard.com/report/github.com/gwos/tcg)
[![GoDoc](https://godoc.org/github.com/gwos/tcg?status.svg)](https://godoc.org/github.com/gwos/tcg)

The Transit Connection Generator (TCG). TCG contains two sub-systems/packages:

1. Transit agent - connects and sends metrics to Groundwork Monitor
2. Controller service - an http server for external control of agent


#### Table of Contents

1. [Dependencies](#dependencies)
2. [Building](#building)
3. [Running](#running)
4. [Docker](#docker)
5. [Testing](#testing)
6. [Environment variables](#envvar)


<a name="dependencies"></a>
## Dependencies

The TCG project is built with Go Modules. See `go.mod` for a list of dependencies. Here are some main frameworks used by this project:

- [NATS Streaming System](nats.io)

    > [About NATS](nats.io/about)

        github.com/nats-io/nats.go
        github.com/nats-io/nats-server

- [Gin Web Framework](github.com/gin-gonic/gin)

    >Gin is a web framework written in Go (Golang).
    It features a martini-like API with much better performance,
    up to 40 times faster.

        github.com/gin-gonic/gin

- [Gin-Swagger](github.com/swaggo/gin-swagger)

    > Gin Gonic middleware to automatically generate RESTful API ocumentation with Swagger 2.0.

        github.com/swaggo/gin-swagger

    > Generate 'docs.go' for Swagger UI

        $ go run github.com/swaggo/swag/cmd/swag@latest init --parseDependency --parseInternal --exclude ./connectors/

    > Swagger url:

        {host}:{port}/swagger/index.html

- [Env](github.com/caarlos0/env)

    > A simple, zero-dependencies library to parse environment variables into structs

        github.com/caarlos0/env

- [Zerolog](github.com/rs/zerolog)

    > The zerolog package provides a fast and simple logger dedicated to JSON output.

        github.com/rs/zerolog


<a name="building"></a>
## Building

```
$ cd tcg
$ go build .
```


### Building C shared library:

```
$ go build -ldflags "-X 'github.com/gwos/tcg/config.buildTime=`date --rfc-3339=s`' -X 'github.com/gwos/tcg/config.buildTag=<TAG>'" -buildmode=c-shared -o libtransit/libtransit.so libtransit/libtransit.go
```

#### or use Makefiles

### Building Connectors:

LINUX:
```
$ CGO_ENABLED=1 go build -v -o tcg -ldflags "-X 'github.com/gwos/tcg/config.buildTime=`date --rfc-3339=s`' -X 'github.com/gwos/tcg/config.buildTag=<TAG>'"
```

OS X:
```
$ CGO_ENABLED=1 go build -v -o tcg -ldflags "-X 'github.com/gwos/tcg/config.buildTime=`date -u +"%Y-%m-%dT%H:%M:%SZ"`' -X 'github.com/gwos/tcg/config.buildTag=<TAG>'"
```

## Building Connectors for OS Targets (Cross Compiling)
```
env GOOS=linux GOARCH=386 CGO_ENABLED=1 go build -v -o tcg -ldflags "-X 'github.com/gwos/tcg/config.buildTime=`date -u +"%Y-%m-%dT%H:%M:%SZ"`' -X 'github.com/gwos/tcg/config.buildTag=<TAG>'"
```
To view supported platforms use commands
```
go tool dist list
go tool dist list -json
go tool dist list -json | jq '.[] | select(.CgoSupported == true)'
```

## Installing as a service
To enable:
```
sudo systemctl enable tcg-elastic
```
To start:
```
sudo systemctl start tcg-elastic
```
Show status:
```
sudo systemctl status tcg-elastic
```
To stop:
```
sudo systemctl stop tcg-elastic
```
To disable:
```
sudo systemctl disable tcg-elastic
```
To reconfigure:
```
sudo systemctl daemon-reload
```
To tail:
```
journalctl -f -u tcg-elastic
```

<a name="running"></a>
## Running

```
$ cd tcg
$ go run .
```


<a name="docker"></a>
## Docker

### Build image:

    $ docker build -t groundworkdevelopment/tcg .


<a name="testing"></a>
## Testing

### Run package tests:

>With logs:

    $ go test -v ./<package_one>/ ./<package_two>/

### Run integration tests:

>For integration tests you must provide environment variables for Groundwork Connection. Also have to deal with TLS certs: get it in local trust storage or suppress check.

    $ TCG_TLS_CLIENT_INSECURE=TRUE \
        TCG_GWCONNECTIONS_0_USERNAME=remote TCG_GWCONNECTIONS_0_PASSWORD=remote \
        go test -v ./integration/


### Examples:

    $ go test -v ./config/ ./services/

    $ go test -v ./libtransit/

    $ go test -v $(go list ./... | grep -v tcg/integration)

    $ GOFLAGS="-count=1" \
        TCG_TLS_CLIENT_INSECURE=TRUE \
        TCG_GWCONNECTIONS_0_USERNAME=remote TCG_GWCONNECTIONS_0_PASSWORD=remote \
        TCG_CONNECTOR_LOGCOLORS=TRUE TCG_CONNECTOR_LOGLEVEL=3 \
        OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317 \
        TCG_CONNECTOR_AGENTID=TEST11 \
        go test -failfast -v ./integration/

    $ GOFLAGS="-count=1" \
        TCG_TLS_CLIENT_INSECURE=TRUE \
        TCG_GWCONNECTIONS_0_USERNAME=remote TCG_GWCONNECTIONS_0_PASSWORD=remote \
        TCG_CONNECTOR_LOGCOLORS=TRUE TCG_CONNECTOR_LOGLEVEL=3 \
        __OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317 \
        __TCG_HTTP_CLIENT_TIMEOUT_GW=120s \
        __TEST_FLAG_CLIENT=true \
        TEST_RESOURCES_COUNT=40 TEST_SERVICES_COUNT=100 \
        TCG_CONNECTOR_BATCHMETRICS=1s TCG_CONNECTOR_BATCHMAXBYTES=204800  TCG_CONNECTOR_NATSMAXPAYLOAD=40920 \
        go test -benchtime=10x -benchmem -run=^$ -bench ^BenchmarkE2E$  ./integration/ \
        | grep _STATS


<a name="envvar"></a>
## Environment variables


### LIBTRANSIT

Defines the path to `libtransit.so` library in docker container and tests.

    $ export LIBTRANSIT=/path/to/libtransit.so


### TCG config variables

By default, the config file is looked for in the work directory as `tcg_config.yaml`.

The path to config file and any config option can be overridden with env vars:

    $ export TCG_CONFIG=/path/to/tcg_config.yaml
    $ export TCG_CONNECTOR_NATSSTORETYPE=MEMORY

For more info see package `config` and tests.


### Other variables

There are additional variables supported:

    * OTEL_EXPORTER_OTLP_ENDPOINT=http://jaegertracing:4317
    * TCG_HTTP_CLIENT_TIMEOUT=10s
    * TCG_HTTP_CLIENT_TIMEOUT_GW=120s
    * TCG_INVENTORY_NOEXT=true
    * TCG_SUPPRESS_DOWNTIMES=true
    * TCG_SUPPRESS_EVENTS=true
    * TCG_SUPPRESS_INVENTORY=true
    * TCG_SUPPRESS_METRICS=true


## Run golangci-lint locally:

    $ go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.54.2
    $ ~/go/bin/golangci-lint --config ./.golangci.yaml run ./... --deadline=2m
