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
	blendedclientset "github.com/inwinstack/blended/client/clientset/versioned/typed/inwinstack/v1"
	opkit "github.com/inwinstack/operator-kit"
	"github.com/inwinstack/pa-operator/pkg/operator/service"
	"github.com/inwinstack/pa-operator/pkg/util/k8sutil"
	"github.com/inwinstack/pa-operator/pkg/util/pautil"
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
	PaloAlto         *pautil.PaloAltoFlag
}

type Operator struct {
	ctx     *opkit.Context
	service *service.ServiceController
	flag    *Flag
}

func NewMainOperator(flag *Flag) *Operator {
	return &Operator{flag: flag}
}

func (o *Operator) Initialize() error {
	glog.V(2).Info("Initialize the operator resources.")

	paclient, err := pautil.NewClient(o.flag.PaloAlto)
	if err != nil {
		return err
	}
	o.showPaloAltoInfors(paclient)

	ctx, blendedclient, err := o.initContextAndClient()
	if err != nil {
		return err
	}

	o.service = service.NewController(ctx, blendedclient, paclient, o.flag.IgnoreNamespaces)
	o.ctx = ctx
	return nil
}

func (o *Operator) showPaloAltoInfors(paclient *pautil.PaloAlto) {
	glog.V(2).Infof("PA version: %s.\n", paclient.GetVersion())
	glog.V(2).Infof("PA hostname: %s.\n", paclient.GetHostname())
	glog.V(2).Infof("PA username: %s.\n", paclient.GetUsername())
}

func (o *Operator) initContextAndClient() (*opkit.Context, blendedclientset.InwinstackV1Interface, error) {
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

	blendedclient, err := blendedclientset.NewForConfig(config)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to create blended clientset. %+v", err)
	}

	ctx := &opkit.Context{
		Clientset:             client,
		APIExtensionClientset: extensionsclient,
		Interval:              interval,
		Timeout:               timeout,
	}
	return ctx, blendedclient, nil
}

func (o *Operator) Run() error {
	signalChan := make(chan os.Signal, 1)
	stopChan := make(chan struct{})
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	// start watching the resources
	o.service.StartWatch(v1.NamespaceAll, stopChan)

	for {
		select {
		case <-signalChan:
			glog.Infof("Shutdown signal received, exiting...")
			close(stopChan)
			return nil
		}
	}
}