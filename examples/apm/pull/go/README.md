<p align="center">
  <a href="http://www.gwos.com/" target="blank"><img src="http://www.gwos.com/wp-content/themes/groundwork/img/gwos_black_orange.png" width="390" alt="GWOS Logo" align="right"/></a>
</p>

# [Prometheus](http://prometheus.io) server example in [Go](http://golang.org)

## Running and integrating

### Prometheus server:

    $ cd examples/prometheus/pull/go
    $ go build
    $ ./go
    
### APM connector:

    $ cd connectors/apm-connector
    $ go build
    $ ./apm-connector

```   
To connect an APM connector to the Golang server, just create a new connector in the UI
and set the resource to `localhost:2222`.
```