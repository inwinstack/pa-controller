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

package pa

import (
	"time"

	"github.com/PaloAltoNetworks/pango"
	"github.com/golang/glog"
	inwinv1 "github.com/inwinstack/blended/apis/inwinstack/v1"
	clientset "github.com/inwinstack/blended/client/clientset/versioned"
	opkit "github.com/inwinstack/operator-kit"
	"github.com/inwinstack/pa-controller/pkg/config"
	"github.com/inwinstack/pa-controller/pkg/operator/pa/nat"
	"github.com/inwinstack/pa-controller/pkg/operator/pa/security"
	"github.com/inwinstack/pa-controller/pkg/operator/pa/service"
	"github.com/inwinstack/pa-controller/pkg/util"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Controller struct {
	ctx       *opkit.Context
	clientset clientset.Interface
	conf      *config.OperatorConfig
	fw        *pango.Firewall
	commit    chan bool

	service  *service.ServiceController
	security *security.SecurityController
	nat      *nat.NATController
}

func NewController(ctx *opkit.Context, clientset clientset.Interface, conf *config.OperatorConfig) *Controller {
	fw := &pango.Firewall{Client: pango.Client{
		Hostname: conf.Host,
		Username: conf.Username,
		Logging:  pango.LogAction | pango.LogOp,
	}}

	if len(conf.Password) != 0 {
		fw.Client.Password = conf.Password
	}

	if len(conf.APIKey) != 0 {
		fw.Client.ApiKey = conf.APIKey
	}

	c := &Controller{
		ctx:       ctx,
		clientset: clientset,
		conf:      conf,
		fw:        fw,
		commit:    make(chan bool),
	}
	return c
}

func (c *Controller) Initialize() error {
	if err := c.fw.Initialize(); err != nil {
		return err
	}

	c.service = service.NewController(c.ctx, c.clientset, c.fw.Objects.Services, c.conf, c.commit)
	c.security = security.NewController(c.ctx, c.clientset, c.fw.Policies.Security, c.conf, c.commit)
	c.nat = nat.NewController(c.ctx, c.clientset, c.fw.Policies.Nat, c.conf, c.commit)
	return nil
}

func (c *Controller) StartWatch(namespace string, stopCh chan struct{}) {
	c.showInfos()
	c.nat.StartWatch(v1.NamespaceAll, stopCh)
	c.security.StartWatch(v1.NamespaceAll, stopCh)
	c.service.StartWatch(v1.NamespaceAll, stopCh)
	go c.handleCommitJob(stopCh)
	go c.keepRetryFailedResources(stopCh)
}

func (c *Controller) showInfos() {
	glog.V(3).Infof("PA version: %s.\n", c.fw.Versioning().String())
	glog.V(3).Infof("PA hostname: %s.\n", c.fw.Hostname)
	glog.V(3).Infof("PA username: %s.\n", c.fw.Username)
}

func (c *Controller) commitToPA() error {
	_, err := c.fw.Commit("", false, true, false, false)
	if err != nil {
		return err
	}
	return nil
}

func (c *Controller) waitNextCommitJob(t time.Duration) bool {
	ch := make(chan struct{})
	go func() {
		<-c.commit
		close(ch)
	}()

	select {
	case <-ch:
		return c.waitNextCommitJob(t)
	case <-time.After(t):
		c.commit <- false
		return true
	}
}

func (c *Controller) handleCommitJob(stopCh <-chan struct{}) {
	for {
		select {
		case ok := <-c.commit:
			if ok {
				run := c.waitNextCommitJob(time.Second * time.Duration(c.conf.CommitWaitTime))
				if run {
					glog.V(3).Infoln("Received commit job signal...")
					util.Retry(c.commitToPA, time.Second*2, c.conf.Retry)
				}
			}
		case <-stopCh:
			return
		}
	}
}

func (c *Controller) keepRetryFailedResources(stopCh <-chan struct{}) {
	interval := time.Duration(c.conf.Interval) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c.retryFailedServices()
			c.retryFailedNATs()
			c.retryFailedSecurities()
		case <-stopCh:
			return
		}
	}
}

func (c *Controller) retryFailedSecurities() {
	secs, err := c.clientset.InwinstackV1().Securities(v1.NamespaceAll).List(metav1.ListOptions{})
	if err != nil {
		glog.Error(err)
	}

	for _, sec := range secs.Items {
		if sec.Status.Phase == inwinv1.SecurityFailed {
			glog.V(4).Infof("Retrying object on Security %s in namespace %s.", sec.Name, sec.Namespace)
			if err := c.setRefresh(&sec); err != nil {
				glog.Errorf("Failed to retry object on Security %s in namespace %s: %+v.", sec.Name, sec.Namespace, err)
			}
		}
	}
}

func (c *Controller) retryFailedNATs() {
	nats, err := c.clientset.InwinstackV1().NATs(v1.NamespaceAll).List(metav1.ListOptions{})
	if err != nil {
		glog.Error(err)
	}

	for _, nat := range nats.Items {
		if nat.Status.Phase == inwinv1.NATFailed {
			glog.V(4).Infof("Retrying object on NAT %s in namespace %s.", nat.Name, nat.Namespace)
			if err := c.setRefresh(&nat); err != nil {
				glog.Errorf("Failed to retry object on NAT %s in namespace %s: %+v.", nat.Name, nat.Namespace, err)
			}
		}
	}
}

func (c *Controller) retryFailedServices() {
	srvcs, err := c.clientset.InwinstackV1().Services().List(metav1.ListOptions{})
	if err != nil {
		glog.Error(err)
	}

	for _, srvc := range srvcs.Items {
		if srvc.Status.Phase == inwinv1.ServiceFailed {
			glog.V(4).Infof("Retrying object on Service %s.", srvc.Name)
			if err := c.setRefresh(&srvc); err != nil {
				glog.Errorf("Failed to retry object on Service %s: %+v.", srvc.Name, err)
			}
		}
	}
}

func (c *Controller) setRefresh(obj interface{}) error {
	switch v := obj.(type) {
	case inwinv1.Security:
		return c.security.SetRefresh(&v)
	case inwinv1.NAT:
		return c.nat.SetRefresh(&v)
	case inwinv1.Service:
		return c.service.SetRefresh(&v)
	}
	return nil
}
