<p align="center">
  <a href="http://www.gwos.com/" target="blank"><img src="https://www.gwos.com/wp-content/themes/groundwork/img/gwos_black_orange.png" width="390" alt="GWOS Logo" align="right"/></a>
</p>

# Tng

The New Groundwork Transit connectors (feeders). TNG contains two sub-systems/packages:

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

The TNG project is built with Go Modules. See `go.mod` for a list of dependencies. Here are some of the main frameworks used by this project:

1. [Gin Web Framework](github.com/gin-gonic/gin)

    >Gin is a web framework written in Go (Golang).
    It features a martini-like API with much better performance,
    up to 40 times faster.

        github.com/gin-gonic/gin

2. [Sessions](github.com/gin-contrib/sessions)

    > Gin middleware for session management with multi-backend support.

        github.com/gin-gonic/contrib/sessions

3. [NATS Streaming System](nats.io)

    > [About NATS](nats.io/about)

        github.com/nats-io/go-nats-streaming
        github.com/nats-io/nats-streaming-server/server
        github.com/nats-io/nats-streaming-server/stores

4. [Envconfig](github.com/kelseyhightower/envconfig)

    > Package envconfig implements decoding of environment variables based
    on a user defined specification. A typical use is using environment variables
    for configuration settings.

        github.com/kelseyhightower/envconfig

5. [Go-Cache](github.com/patrickmn/go-cache)

   > Go-Cache is an in-memory key:value store/cache similar to memcached
    that is suitable for applications running on a single machine. Its major advantage
    is that, being essentially a thread-safe map[string]interface{} with expiration times,
    it doesn't need to serialize or transmit its contents over the network.
    Any object can be stored, for a given duration or forever, and the cache can be safely
    used by multiple goroutines.

        github.com/patrickmn/go-cache

6. [Testify](github.com/stretchr/testify)

    > Go code (golang) set of packages that provide many tools for testifying that your
    code will behave as you intend.

        github.com/stretchr/testify

7. [Logrus](github.com/sirupsen/logrus)

    > Logrus is a structured logger for Go (golang), completely API compatible
    with the standard library logger.

        github.com/sirupsen/logrus
    
    > Log levels:
       
        0 - Error; 
        1 - Warn; 
        2 - Info; 
        3 - Debug


8. [Gopsutil](github.com/shirou/gopsutil)

    > The challenge is porting all psutil functions on some architectures.

        github.com/shirou/gopsutil

9. [Gin-Swagger](github.com/swaggo/gin-swagger)
    
    > Gin Gonic middleware to automatically generate RESTful API documentation with Swagger 2.0.
                                                        
        github.com/swaggo/gin-swagger
        
    > Generate 'docs.go' for Swagger UI
        
        $ swag init
    
    > Swagger url:

        {host}:{port}/swagger/index.html

<a name="building"></a>
## Building

```
$ cd tng
$ go build .
```


### Building tng shared module:

```
$ go build -buildmode=c-shared -o libtransit/libtransit.so libtransit/libtransit.go
```


### or use Makefiles

<a name="running"></a>
## Running

```
$ cd tng
$ go run .
```


<a name="docker"></a>
## Docker

### Build image:

    $ docker build -t groundworkdevelopment/tng .


<a name="testing"></a>
## Testing

The [gotests](https://github.com/cweill/gotests) tool can generate Go tests.


### Run all tests:

>Without logs:

    $ go test ./...

>With logs:

    $ go test -v ./...


### Run package tests:

>Without logs:

    $ go test ./<package_name>/

>With logs:

    $ go test -v ./<package_one>/ ./<package_two>/


### Run tests in Docker container:

>All packages:

    $ ./docker_tests.sh

>One package:

    $ ./docker_tests.sh -v ./<package_one>/ ./<package_two>/

*Available packages:* <b>integration, config, milliseconds, customTime</b>


### Examples:

    $ go test ./integration/

    $ go test -v ./config

    $ ./docker_tests.sh milliseconds


<a name="envvar"></a>
## Environment variables


### LIBTRANSIT

Defines path to `libtransit.so` library in docker container and tests.

    $ export LIBTRANSIT=/path/to/libtransit.so


### TNG

By default the config file is looked for in the work directory as `tng_config.yaml`.

The path to config file and any config option can be overridden with env vars:

    $ export TNG_CONFIG=/path/to/tng_config.yaml
    $ export TNG_CONNECTOR_NATSSTORETYPE=MEMORY

For more info see package `config` and tests.
