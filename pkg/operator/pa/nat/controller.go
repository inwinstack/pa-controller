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

package nat

import (
	"reflect"
	"time"

	"github.com/golang/glog"
	inwinv1 "github.com/inwinstack/blended/apis/inwinstack/v1"
	clientset "github.com/inwinstack/blended/client/clientset/versioned"
	opkit "github.com/inwinstack/operator-kit"
	"github.com/inwinstack/pa-controller/pkg/config"
	"github.com/inwinstack/pa-controller/pkg/constants"
	"github.com/inwinstack/pa-controller/pkg/pautil"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	"github.com/PaloAltoNetworks/pango/poli/nat"
)

const (
	customResourceName       = "nat"
	customResourceNamePlural = "nats"
)

var Resource = opkit.CustomResource{
	Name:    customResourceName,
	Plural:  customResourceNamePlural,
	Group:   inwinv1.CustomResourceGroup,
	Version: inwinv1.Version,
	Scope:   apiextensionsv1beta1.NamespaceScoped,
	Kind:    reflect.TypeOf(inwinv1.NAT{}).Name(),
}

type NATController struct {
	ctx       *opkit.Context
	clientset clientset.Interface
	conf      *config.Operator
	nat       *nat.FwNat
	commit    chan bool
}

func NewController(
	ctx *opkit.Context,
	clientset clientset.Interface,
	nat *nat.FwNat,
	conf *config.Operator,
	commit chan bool) *NATController {
	return &NATController{
		ctx:       ctx,
		clientset: clientset,
		nat:       nat,
		conf:      conf,
		commit:    commit,
	}
}

func (c *NATController) StartWatch(namespace string, stopCh chan struct{}) error {
	resourceHandlerFuncs := cache.ResourceEventHandlerFuncs{
		AddFunc:    c.onAdd,
		UpdateFunc: c.onUpdate,
		DeleteFunc: c.onDelete,
	}

	glog.Infof("Start watching nat resources.")
	watcher := opkit.NewWatcher(Resource, namespace, resourceHandlerFuncs, c.clientset.InwinstackV1().RESTClient())
	go watcher.Watch(&inwinv1.NAT{}, stopCh)
	return nil
}

func (c *NATController) onAdd(obj interface{}) {
	nat := obj.(*inwinv1.NAT).DeepCopy()
	glog.V(2).Infof("Received add on NAT %s in %s namespace.", nat.Name, nat.Namespace)

	if nat.Status.Phase == "" {
		nat.Status.Phase = inwinv1.NATPending
	}

	if nat.Status.Phase == inwinv1.NATPending || nat.Status.Phase == inwinv1.NATFailed {
		if err := c.createOrUpdatePolicy(nat); err != nil {
			glog.Errorf("Failed to set policy on NAT %s in %s namespace: %+v.", nat.Name, nat.Namespace, err)
		}
	}
}

func (c *NATController) onUpdate(oldObj, newObj interface{}) {
	old := oldObj.(*inwinv1.NAT).DeepCopy()
	nat := newObj.(*inwinv1.NAT).DeepCopy()
	glog.V(2).Infof("Received update on NAT %s in %s namespace.", nat.Name, nat.Namespace)

	_, refresh := nat.Annotations[constants.AnnKeyServiceRefresh]
	if !reflect.DeepEqual(old.Spec, nat.Spec) || refresh || nat.Status.Phase == inwinv1.NATPending {
		if err := c.createOrUpdatePolicy(nat); err != nil {
			glog.Errorf("Failed to update policy on NAT %s in %s namespace: %+v.", nat.Name, nat.Namespace, err)
		}
	}
}

func (c *NATController) onDelete(obj interface{}) {
	nat := obj.(*inwinv1.NAT).DeepCopy()
	glog.V(2).Infof("Received delete on NAT %s in %s namespace.", nat.Name, nat.Namespace)

	if err := c.deletePolicy(nat); err != nil {
		glog.Errorf("Failed to delete policy on NAT %s in %s namespace: %+v.", nat.Name, nat.Namespace, err)
	}
}

func (c *NATController) SetRefresh(nat *inwinv1.NAT) error {
	if nat.Annotations == nil {
		nat.Annotations = map[string]string{}
	}

	nat.Annotations[constants.AnnKeyServiceRefresh] = "true"
	if _, err := c.clientset.InwinstackV1().NATs(nat.Namespace).Update(nat); err != nil {
		return err
	}
	return nil
}

func (c *NATController) createOrUpdatePolicy(nat *inwinv1.NAT) error {
	entry := pautil.ToNatEntry(nat)
	if err := c.nat.Edit(c.conf.Vsys, *entry); err != nil {
		return c.createFailedStatus(nat, err)
	}

	// commit the changes to PA
	c.commit <- true

	nat.Status.Phase = inwinv1.NATActive
	nat.Status.Reason = ""
	nat.Status.LastUpdateTime = metav1.NewTime(time.Now())
	delete(nat.Annotations, constants.AnnKeyServiceRefresh)
	if _, err := c.clientset.InwinstackV1().NATs(nat.Namespace).Update(nat); err != nil {
		return err
	}
	return nil
}

func (c *NATController) createFailedStatus(nat *inwinv1.NAT, err error) error {
	nat.Status.Phase = inwinv1.NATFailed
	nat.Status.Reason = err.Error()
	nat.Status.LastUpdateTime = metav1.NewTime(time.Now())
	delete(nat.Annotations, constants.AnnKeyServiceRefresh)
	if _, err := c.clientset.InwinstackV1().NATs(nat.Namespace).Update(nat); err != nil {
		return err
	}
	return nil
}

func (c *NATController) deletePolicy(nat *inwinv1.NAT) error {
	if nat.Status.Phase == inwinv1.NATActive {
		if err := c.nat.Delete(c.conf.Vsys, nat.Name); err != nil {
			return err
		}

		// commit the changes to PA
		c.commit <- true
	}
	return nil
}
