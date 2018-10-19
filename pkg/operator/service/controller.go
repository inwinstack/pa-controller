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

package service

import (
	"reflect"

	"github.com/golang/glog"
	opkit "github.com/inwinstack/operator-kit"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
)

var Resource = opkit.CustomResource{
	Name:    "service",
	Plural:  "services",
	Version: "v1",
	Kind:    reflect.TypeOf(v1.Service{}).Name(),
}

type ServiceController struct {
	ctx *opkit.Context
}

func NewController(ctx *opkit.Context) *ServiceController {
	return &ServiceController{ctx: ctx}
}

func (c *ServiceController) StartWatch(namespace string, stopCh chan struct{}) error {
	resourceHandlerFuncs := cache.ResourceEventHandlerFuncs{
		AddFunc:    c.onAdd,
		UpdateFunc: c.onUpdate,
		DeleteFunc: c.onDelete,
	}

	glog.Info("Start watching service resources.")
	watcher := opkit.NewWatcher(Resource, namespace, resourceHandlerFuncs, c.ctx.Clientset.CoreV1().RESTClient())
	go watcher.Watch(&v1.Service{}, stopCh)
	return nil
}

func (c *ServiceController) onAdd(obj interface{}) {
	glog.Infof("Service resource onAdd: %s.", obj.(*v1.Service).Spec.Type)
}

func (c *ServiceController) onUpdate(oldObj, newObj interface{}) {
	glog.Infof("Service resource onUpdate: %s.", newObj.(*v1.Service).Annotations)
}

func (c *ServiceController) onDelete(obj interface{}) {
	glog.Infof("Service resource onDelete.")
}
