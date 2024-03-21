# networktester

A simple operator to enable self-service network connectivity testing in a Kubernetes cluster.

## Description

Networktester runs as a controller in the Kubernetes cluster. 

It will handle custom resources of type "Networktest" and probe them periodically according to the defined interval.

### Defining the probe

Using **HTTP** probe:

```yaml
apiVersion: edgeworks.no/v1
kind: Networktest
metadata:
  name: vg.no
spec:
  interval: 1m
  timeout: 5
  http:
    url: https://www.vg.no
```

Using **TCP** probe:
```yaml
kind: Networktest
apiVersion: edgeworks.no/v1
metadata:
  name: tcp-success
spec:
  interval: 1m
  timeout: 5
  tcp:
    address: 192.168.0.1
    port: 443
    data: "test" # Optional: Write the data after opening socket
```

**Tip**: Azure Firewall will prevent detection of blocking firewalls if we do not try to send any data 
after opening the socket. The value defined in "data" will be written to the socket after opening. Leave it
empty to disable this feature.

### The probe results are written back to the resource status field.

Success:

```yaml
status:
  accepted: true
  lastResult: Success
  lastRun: "2023-04-24T18:06:23Z"
  message: 192.168.0.1:443
  nextRun: "2023-04-24T18:07:23Z"
```

Failure:

```yaml
status:
  accepted: true
  lastResult: Failed
  lastRun: "2023-04-24T18:06:28Z"
  message: 'timeout: dial tcp 192.168.0.2:443: i/o timeout'
  nextRun: "2023-04-24T18:07:23Z"
```

# Installation

## Container images

Container images are pushed to <https://github.com/edgeworks-as/networktester/pkgs/container/networktester>

Docker pull:

```shell
docker pull ghcr.io/edgeworks-as/networktester:v0.0.1
```

## Helm

Updated charts are pushed to <https://github.com/edgeworks-as/networktester/pkgs/container/networktester%2Fcharts%2Fnetworktester> for each new version. Charts are versioned in line
with the image.

Test templating of chart by doing

```shell
helm template oci://ghcr.io/edgeworks-as/networktester/charts/networktester
```

## Getting Started
You’ll need a Kubernetes cluster to run against. You can use [KIND](https://sigs.k8s.io/kind) to get a local cluster for testing, or run against a remote cluster.
**Note:** Your controller will automatically use the current context in your kubeconfig file (i.e. whatever cluster `kubectl cluster-info` shows).

### How it works
This project aims to follow the Kubernetes [Operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/).

It uses [Controllers](https://kubernetes.io/docs/concepts/architecture/controller/),
which provide a reconcile function responsible for synchronizing resources until the desired state is reached on the cluster.

### Test It Out
1. Install the CRDs into the cluster:

```sh
make install
```

2. Run your controller (this will run in the foreground, so switch to a new terminal if you want to leave it running):

```sh
make run
```

**NOTE:** You can also run this in one step by running: `make install run`

### Modifying the API definitions
If you are editing the API definitions, generate the manifests such as CRs or CRDs using:

```sh
make manifests
```

**NOTE:** Run `make --help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License

Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
