<p align="center">
  <a href="https://www.gwos.com/" target="blank"><img src="https://www.gwos.com/wp-content/themes/groundwork/img/gwos_black_orange.png" width="390" alt="GWOS Logo" align="right"/></a>
</p>

# KUBERNETES CONNECTOR DEPLOYMENT MANUAL

Before begin you need to have the ***kubectl*** command-line tool, that must be configured to communicate with your cluster.
To create a cluster we will use ***[minikube](https://minikube.sigs.k8s.io/docs/start)***. 

>If you don't have the ***kubectl*** command, you can install it as described in the resource below: <br /> <br /> 
> 	• [Install kubectl on Linux](https://kubernetes.io/docs/tasks/tools/install-kubectl-linux) <br />
> 	• [Install kubectl on macOS](https://kubernetes.io/docs/tasks/tools/install-kubectl-macos) <br />
> 	• [Install kubectl on Windows](https://kubernetes.io/docs/tasks/tools/install-kubectl-windows) <br /> <br />

You also need to build [TCG](https://github.com/gwos/tcg) [Docker](https://www.docker.com) image, that we'll use in [Kubernetes](https://kubernetes.io).
See the main [README](https://github.com/gwos/tcg#docker) file for details.

After installing all the necessary tools, just follow the commands below.
From the [TCG](https://github.com/gwos/tcg) project root directory:
```
$ cd connectors/kubernetes-connector
$ kubectl apply -f deployment.yaml
$ kubectl expose deployment kubernetes-connector --type=NodePort --name=kubernetes-connector-service

To restart deployment:
$ kubectl rollout restart deployment kubernetes-connector

To delete deployment:
$ kubectl delete deployment kubernetes-connector

To delete kubctl service:
$ kubectl delete svc kubernetes-connector-service

To make sure the connector works:
$ kubectl logs -l app=kubernetes-connector

Service details:
$ kubectl describe service kubernetes-connector-service

To see minikube dashboard:
$ minikube dashboard

Minikube services details:
$ minikube service list
```

Last command should show the output like:
```
|----------------------|------------------------------|--------------|------------------------------------|
|      NAMESPACE       |             NAME             | TARGET PORT  |                 URL                |
|----------------------|------------------------------|--------------|------------------------------------|
| ...                  |             ...              |         ...  |                 ...                |
| default              | kubernetes-connector-service |         8099 | http://<service_ip>:<service_port> |
| ...                  |             ...              |         ...  |                 ...                |
|----------------------|------------------------------|--------------|------------------------------------|
```

> Use ***URL*** to talk to the kubernetes-connector API. (exmpl: http://<service_ip>:<service_port>/api/v1/config)

> NOTE: **deployment.yaml** file configured with ***TCG_CONNECTOR_CONTROLLERPIN*** environment variable and API doesn't require
> any auth