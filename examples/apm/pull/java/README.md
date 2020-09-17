<p align="center">
  <a href="http://www.gwos.com/" target="blank"><img src="http://www.gwos.com/wp-content/themes/groundwork/img/gwos_black_orange.png" width="390" alt="GWOS Logo" align="right"/></a>
</p>

# [Prometheus](http://prometheus.io) server example in [Java](http://www.java.com)

## Running and integrating

### Prometheus server:

    $ cd examples/prometheus/pull/java
    $ mvn clean install
    $ mvn spring-boot:run
    
### APM connector:

    $ cd connectors/apm-connector
    $ go build
    $ ./apm-connector

```   
To connect a APM connector to the Java server, just create a new connector in the UI
and set the resource to `localhost:8080`.
```