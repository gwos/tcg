<p align="center">
  <a href="http://www.gwos.com/" target="blank"><img src="http://www.gwos.com/wp-content/themes/groundwork/img/gwos_black_orange.png" width="390" alt="GWOS Logo" align="right"/></a>
</p>

# [Prometheus](http://prometheus.io) server example in [JS](http://www.javascript.com)

## Running and integrating

### Prometheus server:

    $ cd examples/prometheus/pull/javascript
    $ npm i
    $ node main.js
    
### APM connector:

    $ cd connectors/apm-connector
    $ go build
    $ ./apm-connector

```   
To connect a Prometheus connector to the JS server, just create a new connector in the UI
and set the resource to `localhost:3000`.
```
