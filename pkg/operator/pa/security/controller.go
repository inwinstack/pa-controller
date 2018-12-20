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
	customResourceName       = "security"
	customResourceNamePlural = "securities"
)

var Resource = opkit.CustomResource{
	Name:       customResourceName,
	Plural:     customResourceNamePlural,
	Group:      inwinv1.CustomResourceGroup,
	Version:    inwinv1.Version,
	Scope:      apiextensionsv1beta1.NamespaceScoped,
	Kind:       reflect.TypeOf(inwinv1.Security{}).Name(),
	ShortNames: []string{"sec"},
}

type SecurityController struct {
	ctx       *opkit.Context
	clientset inwinclientset.InwinstackV1Interface
	conf      *config.OperatorConfig
	paclient  *pautil.PaloAlto
	commit    chan int
}

func NewController(
	ctx *opkit.Context,
	clientset inwinclientset.InwinstackV1Interface,
	paclient *pautil.PaloAlto,
	conf *config.OperatorConfig,
	commit chan int) *SecurityController {
	return &SecurityController{
		ctx:       ctx,
		clientset: clientset,
		paclient:  paclient,
		conf:      conf,
		commit:    commit,
	}
}

func (c *SecurityController) StartWatch(namespace string, stopCh chan struct{}) error {
	resourceHandlerFuncs := cache.ResourceEventHandlerFuncs{
		AddFunc:    c.onAdd,
		UpdateFunc: c.onUpdate,
		DeleteFunc: c.onDelete,
	}

	glog.Infof("Start watching security resources.")
	watcher := opkit.NewWatcher(Resource, namespace, resourceHandlerFuncs, c.clientset.RESTClient())
	go watcher.Watch(&inwinv1.Security{}, stopCh)
	return nil
}

func (c *SecurityController) onAdd(obj interface{}) {
	sec := obj.(*inwinv1.Security).DeepCopy()
	glog.V(2).Infof("Received add on Security %s in %s namespace.", sec.Name, sec.Namespace)

	if sec.Status.Phase == "" {
		sec.Status.Phase = inwinv1.SecurityPending
	}

	if sec.Status.Phase == inwinv1.SecurityPending {
		if err := c.setAndUpdatePolicy(sec); err != nil {
			glog.Errorf("Failed to set policy on Security %s in %s namespace: %+v.", sec.Name, sec.Namespace, err)
		}
	}
}

func (c *SecurityController) onUpdate(oldObj, newObj interface{}) {
	old := oldObj.(*inwinv1.Security).DeepCopy()
	sec := newObj.(*inwinv1.Security).DeepCopy()
	glog.V(2).Infof("Received update on Security %s in %s namespace.", sec.Name, sec.Namespace)

	if !reflect.DeepEqual(old.Spec, sec.Spec) || sec.Status.Phase == inwinv1.SecurityPending {
		if err := c.setAndUpdatePolicy(sec); err != nil {
			glog.Errorf("Failed to update policy on Security %s in %s namespace: %+v.", sec.Name, sec.Namespace, err)
		}
	}
}

func (c *SecurityController) onDelete(obj interface{}) {
	sec := obj.(*inwinv1.Security).DeepCopy()
	glog.V(2).Infof("Received delete on Security %s in %s namespace.", sec.Name, sec.Namespace)

	if err := c.deletePolicy(sec); err != nil {
		glog.Errorf("Failed to delete policy on Security %s in %s namespace: %+v.", sec.Name, sec.Namespace, err)
	}
}

func (c *SecurityController) getRetry(sec *inwinv1.Security) int {
	if v, ok := sec.Annotations[constants.AnnKeyPolicyRetry]; ok {
		retry, _ := strconv.Atoi(v)
		return retry
	}
	return 0
}

func (c *SecurityController) setRetry(sec *inwinv1.Security, retry int) {
	if sec.Annotations == nil {
		sec.Annotations = map[string]string{}
	}
	sec.Annotations[constants.AnnKeyPolicyRetry] = strconv.Itoa(retry)
}

func (c *SecurityController) checkRetry(sec *inwinv1.Security, err error) error {
	retry := c.getRetry(sec)
	switch {
	case retry < c.conf.Retry:
		retry++
		c.setRetry(sec, retry)
	case retry >= c.conf.Retry:
		sec.Status.Phase = inwinv1.SecurityFailed
		delete(sec.Annotations, constants.AnnKeyPolicyRetry)
	}

	sec.Status.Reason = err.Error()
	sec.Status.LastUpdateTime = metav1.NewTime(time.Now())
	if _, serr := c.clientset.Securities(sec.Namespace).Update(sec); serr != nil {
		return serr
	}
	return nil
}

func (c *SecurityController) setAndUpdatePolicy(sec *inwinv1.Security) error {
	entry := pautil.ToSecurityEntry(sec)
	if err := c.paclient.Security.Set(entry); err != nil {
		if serr := c.checkRetry(sec, err); serr != nil {
			return serr
		}
		return err
	}

	if err := c.paclient.Security.Move(c.conf.MoveType, c.conf.MoveRelationRule, entry); err != nil {
		if serr := c.checkRetry(sec, err); serr != nil {
			return serr
		}
		return err
	}

	// commit change to PA
	c.commit <- 1

	sec.Status.Phase = inwinv1.SecurityActive
	sec.Status.LastUpdateTime = metav1.NewTime(time.Now())
	if _, err := c.clientset.Securities(sec.Namespace).Update(sec); err != nil {
		return err
	}
	return nil
}

func (c *SecurityController) deletePolicy(sec *inwinv1.Security) error {
	if sec.Status.Phase == inwinv1.SecurityActive {
		if err := c.paclient.Security.Delete(sec.Name); err != nil {
			return err
		}

		// commit change to PA
		c.commit <- 1
	}
	return nil
}
