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
	"strconv"
	"time"

	"github.com/PaloAltoNetworks/pango/objs/srvc"
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
)

const (
	customResourceName       = "service"
	customResourceNamePlural = "services"
)

var Resource = opkit.CustomResource{
	Name:       customResourceName,
	Plural:     customResourceNamePlural,
	Group:      inwinv1.CustomResourceGroup,
	Version:    inwinv1.Version,
	Scope:      apiextensionsv1beta1.ClusterScoped,
	Kind:       reflect.TypeOf(inwinv1.Service{}).Name(),
	ShortNames: []string{"srvc"},
}

type ServiceController struct {
	conf      *config.OperatorConfig
	ctx       *opkit.Context
	clientset clientset.Interface
	srvc      *srvc.FwSrvc
	commit    chan bool
}

func NewController(
	ctx *opkit.Context,
	clientset clientset.Interface,
	srvc *srvc.FwSrvc,
	conf *config.OperatorConfig,
	commit chan bool) *ServiceController {
	return &ServiceController{
		ctx:       ctx,
		clientset: clientset,
		srvc:      srvc,
		conf:      conf,
		commit:    commit,
	}
}

func (c *ServiceController) StartWatch(namespace string, stopCh chan struct{}) error {
	resourceHandlerFuncs := cache.ResourceEventHandlerFuncs{
		AddFunc:    c.onAdd,
		UpdateFunc: c.onUpdate,
		DeleteFunc: c.onDelete,
	}

	glog.Infof("Start watching service obj resources.")
	watcher := opkit.NewWatcher(Resource, namespace, resourceHandlerFuncs, c.clientset.InwinstackV1().RESTClient())
	go watcher.Watch(&inwinv1.Service{}, stopCh)
	return nil
}

func (c *ServiceController) onAdd(obj interface{}) {
	svc := obj.(*inwinv1.Service).DeepCopy()
	glog.V(2).Infof("Received add on Service %s.", svc.Name)

	if svc.Status.Phase == "" {
		svc.Status.Phase = inwinv1.ServicePending
	}

	if svc.Status.Phase == inwinv1.ServicePending {
		if err := c.setAndUpdateObject(svc); err != nil {
			glog.Errorf("Failed to set object on Service %s: %+v.", svc.Name, err)
		}
	}
}

func (c *ServiceController) onUpdate(oldObj, newObj interface{}) {
	old := oldObj.(*inwinv1.Service).DeepCopy()
	svc := newObj.(*inwinv1.Service).DeepCopy()
	glog.V(2).Infof("Received update on Service %s.", svc.Name)

	_, needCommit := svc.Annotations[constants.AnnKeyServiceRefresh]
	if !reflect.DeepEqual(old.Spec, svc.Spec) || needCommit || svc.Status.Phase == inwinv1.ServicePending {
		if err := c.setAndUpdateObject(svc); err != nil {
			glog.Errorf("Failed to update object on Service %s: %+v.", svc.Name, err)
		}
	}
}

func (c *ServiceController) onDelete(obj interface{}) {
	svc := obj.(*inwinv1.Service).DeepCopy()
	glog.V(2).Infof("Received delete on Service %s.", svc.Name)

	if err := c.deleteObject(svc); err != nil {
		glog.Errorf("Failed to delete object on Service %s: %+v.", svc.Name, err)
	}
}

func (c *ServiceController) getRetry(svc *inwinv1.Service) int {
	if v, ok := svc.Annotations[constants.AnnKeyPolicyRetry]; ok {
		retry, _ := strconv.Atoi(v)
		return retry
	}
	return 0
}

func (c *ServiceController) setRetry(svc *inwinv1.Service, retry int) {
	if svc.Annotations == nil {
		svc.Annotations = map[string]string{}
	}
	svc.Annotations[constants.AnnKeyPolicyRetry] = strconv.Itoa(retry)
}

func (c *ServiceController) checkRetry(svc *inwinv1.Service, err error) error {
	retry := c.getRetry(svc)
	switch {
	case retry < c.conf.Retry:
		retry++
		c.setRetry(svc, retry)
	case retry >= c.conf.Retry:
		svc.Status.Phase = inwinv1.ServiceFailed
		delete(svc.Annotations, constants.AnnKeyPolicyRetry)
	}

	svc.Status.Reason = err.Error()
	svc.Status.LastUpdateTime = metav1.NewTime(time.Now())
	if _, serr := c.clientset.InwinstackV1().Services().Update(svc); serr != nil {
		return serr
	}
	return nil
}

func (c *ServiceController) setAndUpdateObject(svc *inwinv1.Service) error {
	entry := pautil.ToServiceEntry(svc)
	if err := c.srvc.Edit(c.conf.Vsys, *entry); err != nil {
		if serr := c.checkRetry(svc, err); serr != nil {
			return serr
		}
		return err
	}

	// commit the changes to PA
	c.commit <- true

	svc.Status.Phase = inwinv1.ServiceActive
	svc.Status.LastUpdateTime = metav1.NewTime(time.Now())
	delete(svc.Annotations, constants.AnnKeyServiceRefresh)
	if _, err := c.clientset.InwinstackV1().Services().Update(svc); err != nil {
		return err
	}
	return nil
}

func (c *ServiceController) deleteObject(svc *inwinv1.Service) error {
	if svc.Status.Phase == inwinv1.ServiceActive {
		if err := c.srvc.Delete(c.conf.Vsys, svc.Name); err != nil {
			return err
		}

		// commit the changes to PA
		c.commit <- true
	}
	return nil
}
