# podinfo-operator

An operator to deploy [podinfo](https://github.com/stefanprodan/podinfo)

## Getting Started

### Prerequisites
- go version v1.21.0+
- docker version 17.03+.
- kubectl version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.

### To Deploy on the cluster

#### Via Helm Chart

```
cd helm/podinfo-operator
helm upgrade --install podinfo-operator . -f example.values.yaml --namespace=podinfo-operator --create-namespace

```

#### Via Make Targets
```
make install
make deploy IMG=docker.io/patjones/podinfo-operator:240317-150819-main-521beb7
```

### Testing the Operator

#### Installing the Test Object

Once the operator is installed you can deploy a test `MyAppResource` object via the `test-crd.yaml` either in the repo root directory or in the `helm/` subdirectory
```
kubectl create namespace whatever
kubectl apply -f test-crd.yaml
```

#### Testing podinfo
```
// assuming the test crd used is named `whatever` in the `whatever` ns
kubectl port-forward service/whatever -n whatever 8080:80
// navigate to http://localhost:8080/ in browser

// redis testing
curl -X PUT http://localhost:8080/cache/test-key-1 -v
curl -X PUT http://localhost:8080/cache/test-key-2 -v
//print keys in redis
k exec -ti whatever-redis-{pod identifier} -n whatever -- sh -c "redis-cli -n 0 KEYS "*""
```

#### Kubebuilder Built-ins

**Build and push your image to the location specified by `IMG`:**

```sh
make docker-build docker-push IMG=<some-registry>/podinfo-operator:tag
```

**NOTE:** This image ought to be published in the personal registry you specified. 
And it is required to have access to pull the image from the working environment. 
Make sure you have the proper permission to the registry if the above commands don’t work.

**Install the CRDs into the cluster:**

```sh
make install
```

**Deploy the Manager to the cluster with the image specified by `IMG`:**

```sh
make deploy IMG=<some-registry>/podinfo-operator:tag
```

> **NOTE**: If you encounter RBAC errors, you may need to grant yourself cluster-admin 
privileges or be logged in as admin.

**Create instances of your solution**
You can apply the samples (examples) from the config/sample:

```sh
kubectl apply -k config/samples/
```

>**NOTE**: Ensure that the samples has default values to test it out.

### To Uninstall
**Delete the instances (CRs) from the cluster:**

```sh
kubectl delete -k config/samples/
```

**Delete the APIs(CRDs) from the cluster:**

```sh
make uninstall
```

**UnDeploy the controller from the cluster:**

```sh
make undeploy
```

## Project Distribution

Following are the steps to build the installer and distribute this project to users.

1. Build the installer for the image built and published in the registry:

```sh
make build-installer IMG=<some-registry>/podinfo-operator:tag
```

NOTE: The makefile target mentioned above generates an 'install.yaml'
file in the dist directory. This file contains all the resources built
with Kustomize, which are necessary to install this project without
its dependencies.

2. Using the installer

Users can just run kubectl apply -f <URL for YAML BUNDLE> to install the project, i.e.:

```sh
kubectl apply -f https://raw.githubusercontent.com/<org>/podinfo-operator/<tag or branch>/dist/install.yaml
```

## Contributing
// TODO(user): Add detailed information on how you would like others to contribute to this project

**NOTE:** Run `make help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License

Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

