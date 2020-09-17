<p align="center">
  <a href="http://www.gwos.com/" target="blank"><img src="http://www.gwos.com/wp-content/themes/groundwork/img/gwos_black_orange.png" width="390" alt="GWOS Logo" align="right"/></a>
</p>

# [Prometheus](http://prometheus.io) PUSH example in [JS](http://www.javascript.com)

## Running and integrating

### APM connector:

    $ cd connectors/apm-connector
    $ go build
    $ ./apm-connector
    
### Prometheus JavaScript PUSH tool:

    $ cd examples/prometheus/push/javascript
    $ npm i
    $ node main.js {foundation host} {user} {password} {gwos-app-name}
    
> Note, that you need to provide arguments in certain order without '{}' (as shown in the line above)
