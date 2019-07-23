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

package security

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
	"github.com/inwinstack/pango/poli/security"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// Controller represents the controller of security
type Controller struct {
	cfg        *config.Config
	fwSec      *security.FwSecurity
	blendedset blended.Interface
	lister     listerv1.SecurityLister
	synced     cache.InformerSynced
	queue      workqueue.RateLimitingInterface
	commit     chan bool
}

// NewController creates an instance of the security controller
func NewController(
	cfg *config.Config,
	fwSec *security.FwSecurity,
	blendedset blended.Interface,
	informer informerv1.SecurityInformer,
	commit chan bool) *Controller {
	controller := &Controller{
		cfg:        cfg,
		fwSec:      fwSec,
		blendedset: blendedset,
		lister:     informer.Lister(),
		synced:     informer.Informer().HasSynced,
		queue:      workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Securities"),
		commit:     commit,
	}
	informer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueue,
		UpdateFunc: func(old, new interface{}) {
			oo := old.(*blendedv1.Security)
			no := new.(*blendedv1.Security)
			k8sutil.MakeNeedToUpdate(&no.ObjectMeta, oo.Spec, no.Spec)
			controller.enqueue(no)
		},
	})
	return controller
}

// Run serves the security controller
func (c *Controller) Run(ctx context.Context, threadiness int) error {
	glog.Info("Starting the security controller")
	glog.Info("Waiting for the security informer caches to sync")
	if ok := cache.WaitForCacheSync(ctx.Done(), c.synced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, ctx.Done())
	}
	return nil
}

// Stop stops the security controller
func (c *Controller) Stop() {
	glog.Info("Stopping the security controller")
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
			utilruntime.HandleError(fmt.Errorf("Security expected string in workqueue but got %#v", obj))
			return nil
		}

		if err := c.reconcile(key); err != nil {
			c.queue.AddRateLimited(key)
			return fmt.Errorf("Security error syncing '%s': %s, requeuing", key, err.Error())
		}

		c.queue.Forget(obj)
		glog.Infof("Security successfully synced '%s'", key)
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
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return err
	}

	security, err := c.lister.Securities(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("security '%s' in work queue no longer exists", key))
			return err
		}
		return err
	}

	if !security.ObjectMeta.DeletionTimestamp.IsZero() {
		if err := c.cleanup(security); err != nil {
			return err
		}
		return nil
	}

	if err := c.checkAndUdateFinalizer(security); err != nil {
		return err
	}

	need := k8sutil.IsNeedToUpdate(security.ObjectMeta)
	if security.Status.Phase != blendedv1.SecurityActive || need {
		if security.Status.Phase == blendedv1.SecurityFailed {
			t := util.SubtractNowTime(security.Status.LastUpdateTime.Time)
			if t.Seconds() <= float64(c.cfg.SyncSec) && !need {
				return nil
			}
		}
		if err := c.createOrUpdate(security); err != nil {
			return c.makeFailed(security, err)
		}
	}
	return nil
}

func (c *Controller) checkAndUdateFinalizer(sec *blendedv1.Security) error {
	secCopy := sec.DeepCopy()
	ok := funk.ContainsString(secCopy.Finalizers, constants.CustomFinalizer)
	if secCopy.Status.Phase == blendedv1.SecurityActive && !ok {
		k8sutil.AddFinalizer(&secCopy.ObjectMeta, constants.CustomFinalizer)
		if _, err := c.blendedset.InwinstackV1().Securities(secCopy.Namespace).Update(secCopy); err != nil {
			return err
		}
	}
	return nil
}

func (c *Controller) makeFailed(sec *blendedv1.Security, e error) error {
	secCopy := sec.DeepCopy()
	secCopy.Status.Reason = e.Error()
	secCopy.Status.Phase = blendedv1.SecurityFailed
	secCopy.Status.LastUpdateTime = metav1.NewTime(time.Now())
	delete(secCopy.Annotations, constants.NeedUpdateKey)
	if _, err := c.blendedset.InwinstackV1().Securities(secCopy.Namespace).Update(secCopy); err != nil {
		return err
	}
	glog.Errorf("Security got an error:%+v.", e)
	return nil
}

func (c *Controller) createOrUpdate(sec *blendedv1.Security) error {
	secCopy := sec.DeepCopy()
	if err := c.updateSecurityPolicy(secCopy); err != nil {
		return err
	}

	secCopy.Status.Reason = ""
	secCopy.Status.Phase = blendedv1.SecurityActive
	secCopy.Status.LastUpdateTime = metav1.NewTime(time.Now())
	delete(secCopy.Annotations, constants.NeedUpdateKey)
	k8sutil.AddFinalizer(&secCopy.ObjectMeta, constants.CustomFinalizer)
	if _, err := c.blendedset.InwinstackV1().Securities(secCopy.Namespace).Update(secCopy); err != nil {
		return err
	}
	return nil
}

func (c *Controller) cleanup(sec *blendedv1.Security) error {
	secCopy := sec.DeepCopy()
	if err := c.deleteSecurityPolicy(secCopy); err != nil {
		return err
	}

	k8sutil.RemoveFinalizer(&secCopy.ObjectMeta, constants.CustomFinalizer)
	secCopy.Status.Phase = blendedv1.SecurityTerminating
	if _, err := c.blendedset.InwinstackV1().Securities(secCopy.Namespace).Update(secCopy); err != nil {
		return err
	}
	return nil
}
