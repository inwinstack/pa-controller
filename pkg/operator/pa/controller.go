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
	clientset "github.com/inwinstack/blended/client/clientset/versioned"
	opkit "github.com/inwinstack/operator-kit"
	"github.com/inwinstack/pa-controller/pkg/config"
	"github.com/inwinstack/pa-controller/pkg/operator/pa/nat"
	"github.com/inwinstack/pa-controller/pkg/operator/pa/security"
	"github.com/inwinstack/pa-controller/pkg/operator/pa/service"
	"github.com/inwinstack/pa-controller/pkg/util"
	"k8s.io/api/core/v1"
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
	go c.handleCommitJob()
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

func (c *Controller) handleCommitJob() {
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
		}
	}
}
