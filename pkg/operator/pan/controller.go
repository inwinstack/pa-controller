/*
Copyright © 2018 inwinSTACK Inc

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

package pan

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/glog"
	blended "github.com/inwinstack/blended/generated/clientset/versioned"
	blendedinformers "github.com/inwinstack/blended/generated/informers/externalversions"
	"github.com/inwinstack/blended/util"
	"github.com/inwinstack/pa-controller/pkg/config"
	"github.com/inwinstack/pa-controller/pkg/operator/pan/nat"
	"github.com/inwinstack/pa-controller/pkg/operator/pan/security"
	"github.com/inwinstack/pa-controller/pkg/operator/pan/service"
	"github.com/inwinstack/pango"
)

// Controller represents the controller of PAN
type Controller struct {
	cfg      *config.Config
	fw       *pango.Firewall
	service  *service.Controller
	nat      *nat.Controller
	security *security.Controller

	commit chan bool
}

// NewController creates an instance of the PAN controller
func NewController(
	cfg *config.Config,
	fw *pango.Firewall,
	blendedset blended.Interface,
	informer blendedinformers.SharedInformerFactory) *Controller {
	c := &Controller{
		cfg:    cfg,
		fw:     fw,
		commit: make(chan bool, 1),
	}
	c.nat = nat.NewController(cfg, fw.Policies.Nat, blendedset, informer.Inwinstack().V1().NATs(), c.commit)
	c.service = service.NewController(cfg, fw.Objects.Services, blendedset, informer.Inwinstack().V1().Services(), c.commit)
	c.security = security.NewController(cfg, fw.Policies.Security, blendedset, informer.Inwinstack().V1().Securities(), c.commit)
	return c
}

// Run serves the PAN controller
func (c *Controller) Run(ctx context.Context, threadiness int) error {
	glog.Info("Starting the PAN controller")
	go c.handleCommitJob(ctx.Done())

	if err := c.service.Run(ctx, c.cfg.Threads); err != nil {
		return fmt.Errorf("failed to run the service controller: %s", err.Error())
	}

	if err := c.nat.Run(ctx, c.cfg.Threads); err != nil {
		return fmt.Errorf("failed to run the nat controller: %s", err.Error())
	}

	if err := c.security.Run(ctx, c.cfg.Threads); err != nil {
		return fmt.Errorf("failed to run the security controller: %s", err.Error())
	}
	return nil
}

// Stop stops the PAN controller
func (c *Controller) Stop() {
	glog.Info("Stopping the PAN controller")
	c.nat.Stop()
	c.security.Stop()
	c.service.Stop()
}

func (c *Controller) commitToPAN() error {
	_, err := c.fw.Commit(c.cfg.Vsys, c.cfg.Admins, c.cfg.DaNPartial, c.cfg.PaOPartial, c.cfg.Force, c.cfg.Sync)
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
				if c.waitNextCommitJob(time.Second * time.Duration(c.cfg.CommitWaitTime)) {
					glog.V(3).Infoln("Received commit job signal...")
					util.Retry(c.commitToPAN, time.Second*2, c.cfg.Retry)
				}
			}
		case <-stopCh:
			return
		}
	}
}
