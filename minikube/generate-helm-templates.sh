#!/bin/bash

# This script generates output from Helm for TICK components.
apps=(influxdb2 kapacitor )

for app in "${apps[@]}"
do
  echo "Generating template for $app"
  helm template picasa influxdata/$app --namespace picasa-local -f $app/values.yaml -a "networking.k8s.io/v1/Ingress" > $app/template.yaml
done


