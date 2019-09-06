# tng
The New Groundwork Transit connectors (feeders). TNG contains two sub-systems/packages:

1. Transit agent - connects and sends metrics to Groundwork Monitor 
2. Controller service - an http server for external control of agent

Imports
--------
1. github.com/gin-gonic/gin

    • Gin is a web framework written in Go (Golang).
      It features a martini-like API with much better performance,
      up to 40 times faster.
    
        go get github.com/gin-gonic/gin

2. github.com/gin-gonic/contrib/sessions

    • Gin middleware for session management with multi-backend support.

        go get github.com/gin-gonic/contrib/sessions
        
3. github.com/kelseyhightower/envconfig

    • Package envconfig implements decoding of environment variables based 
      on a user defined specification. A typical use is using environment variables
      for configuration settings.
    
        go get github.com/kelseyhightower/envconfig
   
Building
--------
```
cd go/src/github/com/gwos/tng
go build .
```

Running 
--------
```
cd go/src/github/com/gwos/tng
go run .
```