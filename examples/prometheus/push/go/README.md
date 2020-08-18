<p align="center">
  <a href="http://www.gwos.com/" target="blank"><img src="http://www.gwos.com/wp-content/themes/groundwork/img/gwos_black_orange.png" width="390" alt="GWOS Logo" align="right"/></a>
</p>

# [Prometheus](http://prometheus.io) PUSH example in [Go](http://golang.org)

## Running and integrating

### Prometheus connector:

    $ cd connectors/prometheus-connector
    $ go build
    $ ./prometheus-connector
    
### Prometheus PUSH tool:

    $ cd examples/prometheus/push/go
    $ go build
    $ ./go -foundationUrl=http://... -user=RESTAPIUSER -password=**** -gwosAppName=GW8
    
### Prometheus PUSH tool HELP:

    $ ./go -h