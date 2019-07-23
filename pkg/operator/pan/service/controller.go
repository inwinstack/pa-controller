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

package service

import (
	"context"
	"fmt"
	"time"

	"github.com/thoas/go-funk"

	"github.com/golang/glog"
	blendedv1 "github.com/inwinstack/blended/apis/inwinstack/v1"
	"github.com/inwinstack/blended/constants"
	blended "github.com/inwinstack/blended/generated/clientset/versioned"
	informerv1 "github.com/inwinstack/blended/generated/informers/externalversions/inwinstack/v1"
	listerv1 "github.com/inwinstack/blended/generated/listers/inwinstack/v1"
	"github.com/inwinstack/blended/k8sutil"
	"github.com/inwinstack/blended/util"
	"github.com/inwinstack/pa-controller/pkg/config"
	"github.com/inwinstack/pango/objs/srvc"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// Controller represents the controller of service
type Controller struct {
	cfg        *config.Config
	srvc       *srvc.FwSrvc
	blendedset blended.Interface
	lister     listerv1.ServiceLister
	synced     cache.InformerSynced
	queue      workqueue.RateLimitingInterface

	commit chan bool
}

// NewController creates an instance of the service controller
func NewController(
	cfg *config.Config,
	srvc *srvc.FwSrvc,
	blendedset blended.Interface,
	informer informerv1.ServiceInformer,
	commit chan bool) *Controller {
	controller := &Controller{
		cfg:        cfg,
		srvc:       srvc,
		blendedset: blendedset,
		lister:     informer.Lister(),
		synced:     informer.Informer().HasSynced,
		queue:      workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "ServiceObjects"),
		commit:     commit,
	}
	informer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueue,
		UpdateFunc: func(old, new interface{}) {
			oo := old.(*blendedv1.Service)
			no := new.(*blendedv1.Service)
			k8sutil.MakeNeedToUpdate(&no.ObjectMeta, oo.Spec, no.Spec)
			controller.enqueue(no)
		},
	})
	return controller
}

// Run serves the service controller
func (c *Controller) Run(ctx context.Context, threadiness int) error {
	glog.Info("Starting the service controller")
	glog.Info("Waiting for the service informer caches to sync")
	if ok := cache.WaitForCacheSync(ctx.Done(), c.synced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, ctx.Done())
	}
	return nil
}

// Stop stops the service controller
func (c *Controller) Stop() {
	glog.Info("Stopping the service controller")
	c.queue.ShutDown()
}

func (c *Controller) runWorker() {
	defer utilruntime.HandleCrash()
	for c.processNextWorkItem() {
	}
}

func (c *Controller) processNextWorkItem() bool {
	obj, shutdown := c.queue.Get()
	if shutdown {
		return false
	}

	err := func(obj interface{}) error {
		defer c.queue.Done(obj)
		key, ok := obj.(string)
		if !ok {
			c.queue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("Service expected string in workqueue but got %#v", obj))
			return nil
		}

		if err := c.reconcile(key); err != nil {
			c.queue.AddRateLimited(key)
			return fmt.Errorf("Service error syncing '%s': %s, requeuing", key, err.Error())
		}

		c.queue.Forget(obj)
		glog.Infof("Service successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}
	return true
}

func (c *Controller) enqueue(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.queue.Add(key)
}

func (c *Controller) reconcile(key string) error {
	_, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return err
	}

	service, err := c.lister.Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("service '%s' in work queue no longer exists", key))
			return err
		}
		return err
	}

	if !service.ObjectMeta.DeletionTimestamp.IsZero() {
		if err := c.cleanup(service); err != nil {
			return err
		}
		return nil
	}

	if err := c.checkAndUdateFinalizer(service); err != nil {
		return err
	}

	need := k8sutil.IsNeedToUpdate(service.ObjectMeta)
	if service.Status.Phase != blendedv1.ServiceActive || need {
		if service.Status.Phase == blendedv1.ServiceFailed {
			t := util.SubtractNowTime(service.Status.LastUpdateTime.Time)
			if t.Seconds() <= float64(c.cfg.SyncSec) && !need {
				return nil
			}
		}
		if err := c.createOrUpdate(service); err != nil {
			return c.makeFailed(service, err)
		}
	}
	return nil
}

func (c *Controller) checkAndUdateFinalizer(svc *blendedv1.Service) error {
	svcCopy := svc.DeepCopy()
	ok := funk.ContainsString(svcCopy.Finalizers, constants.CustomFinalizer)
	if svc.Status.Phase == blendedv1.ServiceActive && !ok {
		k8sutil.AddFinalizer(&svcCopy.ObjectMeta, constants.CustomFinalizer)
		if _, err := c.blendedset.InwinstackV1().Services().Update(svcCopy); err != nil {
			return err
		}
	}
	return nil
}

func (c *Controller) makeFailed(svc *blendedv1.Service, e error) error {
	svcCopy := svc.DeepCopy()
	svcCopy.Status.Reason = e.Error()
	svcCopy.Status.Phase = blendedv1.ServiceFailed
	svcCopy.Status.LastUpdateTime = metav1.NewTime(time.Now())
	delete(svcCopy.Annotations, constants.NeedUpdateKey)
	if _, err := c.blendedset.InwinstackV1().Services().Update(svcCopy); err != nil {
		return err
	}
	glog.Errorf("Service got an error:%+v.", e)
	return nil
}

func (c *Controller) createOrUpdate(svc *blendedv1.Service) error {
	svcCopy := svc.DeepCopy()
	if err := c.updateServiceObject(svcCopy); err != nil {
		return err
	}

	svcCopy.Status.Reason = ""
	svcCopy.Status.Phase = blendedv1.ServiceActive
	svcCopy.Status.LastUpdateTime = metav1.NewTime(time.Now())
	delete(svcCopy.Annotations, constants.NeedUpdateKey)
	k8sutil.AddFinalizer(&svcCopy.ObjectMeta, constants.CustomFinalizer)
	if _, err := c.blendedset.InwinstackV1().Services().Update(svcCopy); err != nil {
		return err
	}
	return nil
}

func (c *Controller) cleanup(svc *blendedv1.Service) error {
	svcCopy := svc.DeepCopy()
	if err := c.deleteServiceObject(svcCopy); err != nil {
		return err
	}

	k8sutil.RemoveFinalizer(&svcCopy.ObjectMeta, constants.CustomFinalizer)
	svcCopy.Status.Phase = blendedv1.ServiceTerminating
	if _, err := c.blendedset.InwinstackV1().Services().Update(svcCopy); err != nil {
		return err
	}
	return nil
}
