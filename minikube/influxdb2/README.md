* Template generated from helm

```
helm template picasa influxdata/influxdb2 --namespace picasa-local -f values.yaml -a "networking.k8s.io/v1/Ingress" > template.yaml
```
