# tng
The New Groundwork Transit connectors (feeders). TNG contains two sub-systems/packages:

1. Transit agent - connects and sends metrics to Groundwork Monitor 
2. Controller service - an http server for external control of agent

Dependencies
--------
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
                                                            
   
Building
--------
```
$ cd tng
$ go build .
```
***Building tng shared module:***

```
$ go build -buildmode=c-shared -o libtransit/libtransit.so libtransit/libtransit.go
```
***or use Makefiles***

Running 
--------
```
$ cd tng
$ go run .
```

Docker
--------
***Build image:***

    $ docker build -t groundworkdevelopment/tng .

Testing
-------
The [gotests](https://github.com/cweill/gotests) tool can generate Go tests.

***Run all tests:***
>Without logs:

    $ go test ./...

>With logs:

    $ go test -v ./...

***Run package tests:***
>Without logs:

    $ go test ./<package_name>/
    
>With logs: 
    
    $ go test -v ./<package_name>/
    
***Run tests in Docker container:***
>All packages:

    $ ./docker_tests.sh
    
>One package:
    
    $ ./docker_tests.sh <package_name>
    
*Available packages:* <b>integration, config, milliseconds, customTime

***Examples:***

    $ go test ./integration/
    
    $ go test -v ./config
    
    $ ./docker_tests.sh milliseconds
