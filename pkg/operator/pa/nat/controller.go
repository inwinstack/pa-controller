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
	"strconv"
	"time"

	"github.com/golang/glog"
	inwinv1 "github.com/inwinstack/blended/apis/inwinstack/v1"
	inwinclientset "github.com/inwinstack/blended/client/clientset/versioned/typed/inwinstack/v1"
	opkit "github.com/inwinstack/operator-kit"
	"github.com/inwinstack/pa-controller/pkg/config"
	"github.com/inwinstack/pa-controller/pkg/constants"
	"github.com/inwinstack/pa-controller/pkg/pautil"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
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
	clientset inwinclientset.InwinstackV1Interface
	conf      *config.OperatorConfig
	paclient  *pautil.PaloAlto
	commit    chan bool
}

func NewController(
	ctx *opkit.Context,
	clientset inwinclientset.InwinstackV1Interface,
	paclient *pautil.PaloAlto,
	conf *config.OperatorConfig,
	commit chan bool) *NATController {
	return &NATController{
		ctx:       ctx,
		clientset: clientset,
		paclient:  paclient,
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
	watcher := opkit.NewWatcher(Resource, namespace, resourceHandlerFuncs, c.clientset.RESTClient())
	go watcher.Watch(&inwinv1.NAT{}, stopCh)
	return nil
}

func (c *NATController) onAdd(obj interface{}) {
	nat := obj.(*inwinv1.NAT).DeepCopy()
	glog.V(2).Infof("Received add on NAT %s in %s namespace.", nat.Name, nat.Namespace)

	if nat.Status.Phase == "" {
		nat.Status.Phase = inwinv1.NATPending
	}

	if nat.Status.Phase == inwinv1.NATPending {
		if err := c.setAndUpdatePolicy(nat); err != nil {
			glog.Errorf("Failed to set policy on NAT %s in %s namespace: %+v.", nat.Name, nat.Namespace, err)
		}
	}
}

func (c *NATController) onUpdate(oldObj, newObj interface{}) {
	old := oldObj.(*inwinv1.NAT).DeepCopy()
	nat := newObj.(*inwinv1.NAT).DeepCopy()
	glog.V(2).Infof("Received update on NAT %s in %s namespace.", nat.Name, nat.Namespace)

	if !reflect.DeepEqual(old.Spec, nat.Spec) || nat.Status.Phase == inwinv1.NATPending {
		if err := c.setAndUpdatePolicy(nat); err != nil {
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

func (c *NATController) getRetry(n *inwinv1.NAT) int {
	if v, ok := n.Annotations[constants.AnnKeyPolicyRetry]; ok {
		retry, _ := strconv.Atoi(v)
		return retry
	}
	return 0
}

func (c *NATController) setRetry(n *inwinv1.NAT, retry int) {
	if n.Annotations == nil {
		n.Annotations = map[string]string{}
	}
	n.Annotations[constants.AnnKeyPolicyRetry] = strconv.Itoa(retry)
}

func (c *NATController) setAndUpdatePolicy(n *inwinv1.NAT) error {
	entry := pautil.ToNatEntry(n)
	if err := c.paclient.NAT.Set(entry); err != nil {
		retry := c.getRetry(n)
		switch {
		case retry < c.conf.Retry:
			retry++
			c.setRetry(n, retry)
		case retry >= c.conf.Retry:
			n.Status.Phase = inwinv1.NATFailed
			delete(n.Annotations, constants.AnnKeyPolicyRetry)
		}

		n.Status.Reason = err.Error()
		n.Status.LastUpdateTime = metav1.NewTime(time.Now())
		if _, serr := c.clientset.NATs(n.Namespace).Update(n); serr != nil {
			return serr
		}
		return err
	}

	// commit the changes to PA
	c.commit <- true

	n.Status.Phase = inwinv1.NATActive
	n.Status.LastUpdateTime = metav1.NewTime(time.Now())
	if _, err := c.clientset.NATs(n.Namespace).Update(n); err != nil {
		return err
	}
	return nil
}

func (c *NATController) deletePolicy(n *inwinv1.NAT) error {
	if n.Status.Phase == inwinv1.NATActive {
		if err := c.paclient.NAT.Delete(n.Name); err != nil {
			return err
		}

		// commit the changes to PA
		c.commit <- true
	}
	return nil
}
