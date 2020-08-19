<p align="center">
  <a href="http://www.gwos.com/" target="blank"><img src="http://www.gwos.com/wp-content/themes/groundwork/img/gwos_black_orange.png" width="390" alt="GWOS Logo" align="right"/></a>
</p>

# [Prometheus](http://prometheus.io) PUSH example in [Java](http://www.java.com)

## Running and integrating

### Prometheus connector:

    $ cd connectors/prometheus-connector
    $ go build
    $ ./prometheus-connector
    
### Prometheus Java PUSH tool:

    $ cd examples/prometheus/push/java
    $ mvn -q clean compile exec:java -Dexec.mainClass="Main" -Dexec.args="{foundation host} {user} {password} {gwos-app-name}"
    
> Note, that you need to provide arguments in certain order (as shown in the line above)
