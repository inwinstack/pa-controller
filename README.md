[![Build Status](https://travis-ci.org/inwinstack/pa-operator.svg?branch=master)](https://travis-ci.org/inwinstack/pa-operator) [![Docker Build Status](https://img.shields.io/docker/build/inwinstack/pa-operator.svg)](https://hub.docker.com/r/inwinstack/pa-operator/) [![codecov](https://codecov.io/gh/inwinstack/pa-operator/branch/master/graph/badge.svg)](https://codecov.io/gh/inwinstack/pa-operator) ![Hex.pm](https://img.shields.io/hexpm/l/plug.svg)
# PA Operator
The PA operator will be sync Kubernetes service that makes it easy to set the PA policy.

![](images/architecture.png)

## Building from Source
Clone repo into your go path under `$GOPATH/src`:
```sh
$ git clone https://github.com/inwinstack/pa-operator.git $GOPATH/src/github.com/inwinstack/pa-operator
$ cd $GOPATH/src/github.com/inwinstack/pa-operator
$ make dep
$ make
```

## Debug out of the cluster
Run the following command to debug:
```sh
$ go run cmd/main.go \
    --kubeconfig $HOME/.kube/config \
    --logtostderr \
    --pa-host=172.22.132.114 \
    --pa-username=admin \
    --pa-password=admin \
    --ignore-namespaces=kube-system,default,kube-public
    -v=2
```

## Deploy in the cluster
Run the following command to deploy operator:
```sh
$ kubectl apply -f deploy/
$ kubectl -n kube-system get po -l app=pa-operator
```
