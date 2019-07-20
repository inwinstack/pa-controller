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

package operator

import (
	"context"
	"fmt"
	"time"

	blended "github.com/inwinstack/blended/client/clientset/versioned"
	blendedinformers "github.com/inwinstack/blended/client/informers/externalversions"
	"github.com/inwinstack/pa-controller/pkg/config"
	"github.com/inwinstack/pa-controller/pkg/operator/pan"
	"github.com/inwinstack/pango"
)

const defaultSyncTime = time.Second * 30

// Operator represents an operator context
type Operator struct {
	clientset      blended.Interface
	informer       blendedinformers.SharedInformerFactory
	cfg            *config.Config
	mainController *pan.Controller
}

// New creates an instance of the operator
func New(cfg *config.Config, fw *pango.Firewall, clientset blended.Interface) *Operator {
	t := defaultSyncTime
	if cfg.SyncSec > 0 {
		t = time.Second * time.Duration(cfg.SyncSec)
	}

	o := &Operator{cfg: cfg, clientset: clientset}
	o.informer = blendedinformers.NewSharedInformerFactory(clientset, t)
	o.mainController = pan.NewController(cfg, fw, clientset, o.informer)
	return o
}

// Run serves an isntance of the operator
func (o *Operator) Run(ctx context.Context) error {
	go o.informer.Start(ctx.Done())
	if err := o.mainController.Run(ctx, o.cfg.Threads); err != nil {
		return fmt.Errorf("failed to run main controller: %s", err.Error())
	}
	return nil
}

// Stop stops the main controller
func (o *Operator) Stop() {
	o.mainController.Stop()
}
