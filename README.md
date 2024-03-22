[![Release version](https://github.com/edgeworks-as/networktester/actions/workflows/main.yml/badge.svg)](https://github.com/edgeworks-as/networktester/actions/workflows/main.yml)

# Networktester

A simple operator to enable self-service network connectivity testing in a Kubernetes cluster.

## Usage

Networktester runs as a controller in the Kubernetes cluster. 

It will handle custom resources of type "Networktest" and probe them periodically according to the defined interval. Unless
deployed in single namespace mode, the controller will handle Networktests across all namespaces.

The tests will be performed from the controller itself, and which means it will reflect network connectivity from the controller
namespace, and not necessarily what is the reality in the namespace of a given Networktest CR. This can be fixed by running the
controller in a sngle-namespace mode, and deploy it to specific namespaces.

### Defining tests

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

## Installation

### Container images

Container images are pushed to GitHub Container registry.

The images can be found [here](https://github.com/edgeworks-as/networktester/pkgs/container/networktester).

### Helm

The easiest installation method is through the use of Helm.

Updated charts are pushed to [GitHub Container Registry](https://github.com/edgeworks-as/networktester/pkgs/container/networktester%2Fcharts%2Fnetworktester). 

Charts are versioned in line with the corresponding image version.

Test templating of chart by doing

```shell
helm template oci://ghcr.io/edgeworks-as/networktester/charts/networktester
```

#### Running Networktester in single-namespace mode

Override the restrictNamespace in values.yaml to restrict watching of Networktests to a single namespace. While not necessary, 
it would be a good idea to run the controller in the same namespace.

```shell
helm template oci://ghcr.io/edgeworks-as/networktester/charts/networktester --set restrictNamespace="test"
```

## Development

### Local development

A local development environment is easily set up using Kind and Tilt.

```shell
# Create Kind cluster
./hack/kind.sh

# Deploy using Tilt
tilt up
```

### Testing

End-to-end tests are written in [Chainsaw](https://kyverno.github.io/chainsaw/latest/intro/).

```shell
# Run test suite - after setting up local development environment described above
chainsaw test
```

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
