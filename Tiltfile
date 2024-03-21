
#load('ext://helm_resource', 'helm_resource', 'helm_repo')
#helm_repo('bitnami', 'https://charts.bitnami.com/bitnami')
#helm_resource('mysql', 'bitnami/mysql', resource_deps=['bitnami'])

# generate the controller manifest and deploy to the cluster
# k8s_yaml(kustomize("./config/default"))

k8s_yaml(helm(
    './chart/networktester',
    name='networktester',
    set=[   'image.repository=localhost:5005/networktester',
            'image.tag=latest',
            'serviceMonitor.create=false',
            'restrictNamespace=default',
    ],
))

# build and push the controller image to the local registry
local_resource(
  'go-compile',
  'CGO_ENABLED=0 GOOS=linux go build -o manager main.go',
  deps=['./main.go', './pkg/', './api/'],
  resource_deps = ['deploy'])

docker_build("localhost:5005/networktester", ".", dockerfile='hack/Dockerfile.tilt')
