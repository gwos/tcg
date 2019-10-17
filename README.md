# tng
The New Groundwork Transit connectors (feeders). TNG contains two sub-systems/packages:

1. Transit agent - connects and sends metrics to Groundwork Monitor 
2. Controller service - an http server for external control of agent

Imports
--------
1. [Gin Web Framework](github.com/gin-gonic/gin)

     >Gin is a web framework written in Go (Golang).
      It features a martini-like API with much better performance,
      up to 40 times faster.
    
        $ go get github.com/gin-gonic/gin

2. [Sessions](github.com/gin-contrib/sessions)

    > Gin middleware for session management with multi-backend support.

        $ go get github.com/gin-gonic/contrib/sessions
        
3. [NATS Streaming System](nats.io)
    
    > [About NATS](nats.io/about)
   
        $ go get github.com/nats-io/go-nats-streaming \
                 github.com/nats-io/nats-streaming-server/server \
                 github.com/nats-io/nats-streaming-server/stores
        
4. [Envconfig](github.com/kelseyhightower/envconfig)

    > Package envconfig implements decoding of environment variables based 
      on a user defined specification. A typical use is using environment variables
      for configuration settings.
    
        $ go get github.com/kelseyhightower/envconfig
                                                            
Installing dependencies
--------

> Run `dep ensure` to ensure `vendor/` is in the correct state for your configuration.

```
$ dep ensure
```

> Dep's *solver* regenerates `Gopkg.lock` if it detects any change in your code imports and/or `Gopkg.toml`. If this is
the case, then the new `Gopkg.lock` file is used to completely rewrite `vendor/`.
   
Building
--------
```
$ cd go/src/github/com/gwos/tng
$ go build .
```
#####Building tng shared module:

```
$ go build -buildmode=c-shared -o libtransit/libtransit.so libtransit/libtransit.go
```
######or use Makefiles

Running 
--------
```
$ cd go/src/github/com/gwos/tng
$ go run .
```
