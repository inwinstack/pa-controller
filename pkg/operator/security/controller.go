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

package security

import (
	"reflect"

	"github.com/PaloAltoNetworks/pango"
	"github.com/golang/glog"
	opkit "github.com/inwinstack/operator-kit"
	inwinalphav1 "github.com/inwinstack/pan-operator/pkg/apis/inwinstack/v1alpha1"
	inwinclientset "github.com/inwinstack/pan-operator/pkg/client/clientset/versioned/typed/inwinstack/v1alpha1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/client-go/tools/cache"
)

const (
	customResourceName       = "securitypolicy"
	customResourceNamePlural = "securitypolicies"
)

var Resource = opkit.CustomResource{
	Name:       customResourceName,
	Plural:     customResourceNamePlural,
	Group:      inwinalphav1.CustomResourceGroup,
	Version:    inwinalphav1.Version,
	Scope:      apiextensionsv1beta1.NamespaceScoped,
	Kind:       reflect.TypeOf(inwinalphav1.SecurityPolicy{}).Name(),
	ShortNames: []string{"sp"},
}

type SecurityController struct {
	ctx       *opkit.Context
	paclient  *pango.Firewall
	clientset inwinclientset.InwinstackV1alpha1Interface
}

func NewController(ctx *opkit.Context, clientset inwinclientset.InwinstackV1alpha1Interface, paclient *pango.Firewall) *SecurityController {
	return &SecurityController{ctx: ctx, clientset: clientset, paclient: paclient}
}

func (c *SecurityController) StartWatch(namespace string, stopCh chan struct{}) error {
	resourceHandlerFuncs := cache.ResourceEventHandlerFuncs{
		AddFunc:    c.onAdd,
		UpdateFunc: c.onUpdate,
		DeleteFunc: c.onDelete,
	}

	glog.Info("Start watching security resources.")
	watcher := opkit.NewWatcher(Resource, namespace, resourceHandlerFuncs, c.clientset.RESTClient())
	go watcher.Watch(&inwinalphav1.SecurityPolicy{}, stopCh)
	return nil
}

func (c *SecurityController) onAdd(obj interface{}) {
	glog.Infof("Security resource onAdd.")
}

func (c *SecurityController) onUpdate(oldObj, newObj interface{}) {
	glog.Infof("Security resource onUpdate.")
}

func (c *SecurityController) onDelete(obj interface{}) {
	glog.Infof("Security resource onDelete.")
}
