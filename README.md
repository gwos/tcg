# tng
The New Groundwork Transit connectors (feeders). TNG contains two sub-systems/packages:

1. Transit agent - connects and sends metrics to Groundwork Monitor 
2. Controller service - an http server for external control of agent
 
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


