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

	"github.com/golang/glog"
	inwinclientset "github.com/inwinstack/blended/client/clientset/versioned/typed/inwinstack/v1"
	opkit "github.com/inwinstack/operator-kit"
	"github.com/inwinstack/pa-controller/pkg/config"
	"github.com/inwinstack/pa-controller/pkg/operator/pa/nat"
	"github.com/inwinstack/pa-controller/pkg/operator/pa/security"
	"github.com/inwinstack/pa-controller/pkg/operator/pa/service"
	"github.com/inwinstack/pa-controller/pkg/pautil"
	"github.com/inwinstack/pa-controller/pkg/util"
	"k8s.io/api/core/v1"
)

type PAController struct {
	ctx       *opkit.Context
	clientset inwinclientset.InwinstackV1Interface
	conf      *config.OperatorConfig
	paclient  *pautil.PaloAlto
	commit    chan int

	service  *service.ServiceController
	security *security.SecurityController
	nat      *nat.NATController
}

func NewController(
	ctx *opkit.Context,
	clientset inwinclientset.InwinstackV1Interface,
	paclient *pautil.PaloAlto,
	conf *config.OperatorConfig) *PAController {
	c := &PAController{
		ctx:       ctx,
		clientset: clientset,
		paclient:  paclient,
		conf:      conf,
		commit:    make(chan int, 1),
	}

	c.service = service.NewController(ctx, clientset, paclient, conf, c.commit)
	c.security = security.NewController(ctx, clientset, paclient, conf, c.commit)
	c.nat = nat.NewController(ctx, clientset, paclient, conf, c.commit)
	return c
}

func (c *PAController) StartWatch(namespace string, stopCh chan struct{}) {
	go c.handleCommitJob()
	c.nat.StartWatch(v1.NamespaceAll, stopCh)
	c.security.StartWatch(v1.NamespaceAll, stopCh)
	c.service.StartWatch(v1.NamespaceAll, stopCh)
}

func (c *PAController) handleCommitJob() {
	for {
		select {
		case <-c.commit:
			run := c.waitNextCommitJob(c.commit, time.Second*time.Duration(c.conf.CommitWaitTime))
			if run {
				glog.V(3).Infoln("Received commit job signal...")
				util.Retry(c.paclient.Commit, time.Second*1, c.conf.Retry)
			}
		}
	}
}

func (c *PAController) waitNextCommitJob(commit chan int, t time.Duration) bool {
	ch := make(chan struct{})
	go func() {
		<-commit
		close(ch)
	}()

	select {
	case <-ch:
		return c.waitNextCommitJob(commit, t)
	case <-time.After(t):
		return true
	}
}
