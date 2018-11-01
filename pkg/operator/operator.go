/*
Copyright © 2018 inwinSTACK.inc

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package operator

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang/glog"
	inwinclientset "github.com/inwinstack/blended/client/clientset/versioned/typed/inwinstack/v1"
	opkit "github.com/inwinstack/operator-kit"
	"github.com/inwinstack/pa-operator/pkg/k8sutil"
	"github.com/inwinstack/pa-operator/pkg/operator/nat"
	"github.com/inwinstack/pa-operator/pkg/operator/security"
	"github.com/inwinstack/pa-operator/pkg/operator/service"
	"github.com/inwinstack/pa-operator/pkg/pautil"
	"github.com/inwinstack/pa-operator/pkg/util"
	"k8s.io/api/core/v1"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/kubernetes"
)

const (
	initRetryDelay = 10 * time.Second
	interval       = 500 * time.Millisecond
	timeout        = 60 * time.Second
)

type Flag struct {
	Kubeconfig       string
	IgnoreNamespaces []string
	Retry            int
	CommitWaitTime   int
	PaloAlto         *pautil.Flag
}

type Operator struct {
	ctx       *opkit.Context
	flag      *Flag
	resources []opkit.CustomResource

	// Resource controllers
	service  *service.ServiceController
	security *security.SecurityController
	nat      *nat.NATController

	// PA resources
	paclient *pautil.PaloAlto
	commit   chan int
	count    int
}

func NewMainOperator(flag *Flag) *Operator {
	return &Operator{
		resources: []opkit.CustomResource{nat.Resource, security.Resource},
		flag:      flag,
		commit:    make(chan int, 1),
		count:     0,
	}
}

func (o *Operator) Initialize() error {
	glog.V(2).Info("Initialize the operator resources.")

	paclient, err := pautil.NewClient(o.flag.PaloAlto)
	if err != nil {
		return err
	}
	o.showPaloAltoInfos(paclient)

	ctx, inwinclient, err := o.initContextAndClient()
	if err != nil {
		return err
	}

	o.service = service.NewController(ctx, inwinclient, paclient, o.flag.IgnoreNamespaces, o.commit)
	o.security = security.NewController(ctx, inwinclient, paclient, o.flag.Retry, o.commit)
	o.nat = nat.NewController(ctx, inwinclient, paclient, o.flag.Retry, o.commit)
	o.ctx = ctx
	o.paclient = paclient
	return nil
}

func (o *Operator) showPaloAltoInfos(paclient *pautil.PaloAlto) {
	glog.V(2).Infof("PA version: %s.\n", paclient.GetVersion())
	glog.V(2).Infof("PA hostname: %s.\n", paclient.GetHostname())
	glog.V(2).Infof("PA username: %s.\n", paclient.GetUsername())
}

func (o *Operator) initContextAndClient() (*opkit.Context, inwinclientset.InwinstackV1Interface, error) {
	glog.V(2).Info("Initialize the operator context and client.")

	config, err := k8sutil.GetRestConfig(o.flag.Kubeconfig)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to get Kubernetes config. %+v", err)
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to get Kubernetes client. %+v", err)
	}

	extensionsclient, err := apiextensionsclientset.NewForConfig(config)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to create Kubernetes API extension clientset. %+v", err)
	}

	inwinclient, err := inwinclientset.NewForConfig(config)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to create inwinstack clientset. %+v", err)
	}

	ctx := &opkit.Context{
		Clientset:             client,
		APIExtensionClientset: extensionsclient,
		Interval:              interval,
		Timeout:               timeout,
	}
	return ctx, inwinclient, nil
}

func (o *Operator) initResources() error {
	glog.V(2).Info("Initialize the CRD resources.")

	ctx := opkit.Context{
		Clientset:             o.ctx.Clientset,
		APIExtensionClientset: o.ctx.APIExtensionClientset,
		Interval:              interval,
		Timeout:               timeout,
	}

	if err := opkit.CreateCustomResources(ctx, o.resources); err != nil {
		return fmt.Errorf("Failed to create custom resource. %+v", err)
	}
	return nil
}

func (o *Operator) handleCommitJob() {
	for {
		select {
		case <-o.commit:
			run := waitNextCommitJob(o.commit, time.Second*time.Duration(o.flag.CommitWaitTime))
			if run {
				glog.V(3).Infoln("Received commit job signal...")
				util.Retry(o.paclient.Commit, time.Second*1, o.flag.Retry)
			}
		}
	}
}

func waitNextCommitJob(commit chan int, t time.Duration) bool {
	ch := make(chan struct{})
	go func() {
		<-commit
		close(ch)
	}()

	select {
	case <-ch:
		return waitNextCommitJob(commit, t)
	case <-time.After(t):
		return true
	}
}

func (o *Operator) Run() error {
	for {
		err := o.initResources()
		if err == nil {
			break
		}
		glog.Errorf("Failed to init resources. %+v. retrying...", err)
		<-time.After(initRetryDelay)
	}

	signalChan := make(chan os.Signal, 1)
	stopChan := make(chan struct{})
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	// start watching the resources
	o.service.StartWatch(v1.NamespaceAll, stopChan)
	o.nat.StartWatch(v1.NamespaceAll, stopChan)
	o.security.StartWatch(v1.NamespaceAll, stopChan)

	// start receiving the commit job
	go o.handleCommitJob()

	for {
		select {
		case <-signalChan:
			glog.Infof("Shutdown signal received, exiting...")
			close(stopChan)
			return nil
		}
	}
}
