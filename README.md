[![Build Status](https://travis-ci.org/inwinstack/pan-operator.svg?branch=master)](https://travis-ci.org/inwinstack/pan-operator) [![Docker Build Status](https://img.shields.io/docker/build/inwinstack/pan-operator.svg)](https://hub.docker.com/r/inwinstack/pan-operator/) ![Hex.pm](https://img.shields.io/hexpm/l/plug.svg)
# PAN Operator
Firewall and NAT provides a Kubernetes custom resource that makes it easy to set and sync policies on PAN-OS.

## Building from Source
Clone repo into your go path under `$GOPATH/src`:
```sh
$ git clone https://github.com/inwinstack/pan-operator.git $GOPATH/src/github.com/inwinstack/pan-operator
$ cd $GOPATH/src/github.com/inwinstack/pan-operator
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
    --pa-password=admin
```

## Deploy in the cluster
Run the following command to deploy operator:
```sh
$ kubectl apply -f deploy/
$ kubectl -n kube-system get po
```
