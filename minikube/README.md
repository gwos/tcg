# Local Development Infra

  This directory structure has the tools and scripts to run the necessary 
  development infrastructure locally in a minikube environment.
  
## Prerequisites
* virtualbox (https://www.virtualbox.org/)
* minikube (https://minikube.sigs.k8s.io/docs/start/)
* kubectl (https://kubernetes.io/docs/tasks/tools/) 
* kube-ns (https://github.com/ahmetb/kubectx/blob/master/kubens)
* kustomize (https://kustomize.io/)
* jq (https://stedolan.github.io/jq/)
* yq (https://github.com/mikefarah/yq)


Many of these are available through homebrew on both OSX and Linux.  

## Minikube Setup
Minikube can be setup many ways, but the way this guide is going to 
describe is to make use of VirtualBox to house the minikube instance.

Once you have minikube installed, you can fire up a cluster with a command like this on linux or osx:
```
minikube start --driver=virtualbox \
  --addons=ingress,metrics-server,storage-provisioner,default-storageclass \
  --cpus=6 --memory=8g --disk-size=50g
```
or something like this on Windows:
```
minikube start --vm-driver=hyperv \
  --addons=ingress,metrics-server,storage-provisioner,default-storageclass \
  --cpus=6 --memory=10g --disk-size=50g --hyperv-virtual-switch="Bridge Switch"
```
The addons are required to make use of the kustomize manifests for development.

Once your minikube cluster is started, you should be able to run `minikube ip` and see 
the address it's using. You can run `echo $(minikube ip) influxdb.local kapacitor.local`
to get an entry to put in /etc/hosts to allow local access. This isn't automatic, because
making assumptions about a developers custom hosts file is bad. (: 

Any shell scripts or local development configs will reference these endpoints as needed.

## Bucket Management
Buckets can be created/deleted with the following commands
```
# Create (24h data TTL) 
kubectl exec -it picasa-influxdb2-0 -- bash -c 'influx bucket create -n raw -o influxdata -r 24h'
kubectl exec -it picasa-influxdb2-0 -- bash -c 'influx bucket create -n raw -o influxdata -r 24h'

# Delete
kubectl exec -it picasa-influxdb2-0 -- bash -c 'influx bucket delete -n raw -o influxdata'
kubectl exec -it picasa-influxdb2-0 -- bash -c 'influx bucket delete -n raw -o influxdata'
```
