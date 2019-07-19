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

package nat

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/glog"
	blendedv1 "github.com/inwinstack/blended/apis/inwinstack/v1"
	blended "github.com/inwinstack/blended/client/clientset/versioned"
	informerv1 "github.com/inwinstack/blended/client/informers/externalversions/inwinstack/v1"
	listerv1 "github.com/inwinstack/blended/client/listers/inwinstack/v1"
	"github.com/inwinstack/pa-controller/pkg/config"
	"github.com/inwinstack/pa-controller/pkg/constants"
	"github.com/inwinstack/pa-controller/pkg/k8sutil"
	"github.com/inwinstack/pa-controller/pkg/util"
	"github.com/inwinstack/pango/poli/nat"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// Controller represents the controller of nat
type Controller struct {
	cfg        *config.Config
	fwNat      *nat.FwNat
	blendedset blended.Interface
	lister     listerv1.NATLister
	synced     cache.InformerSynced
	queue      workqueue.RateLimitingInterface

	commit chan bool
}

// NewController creates an instance of the nat controller
func NewController(
	cfg *config.Config,
	fwNat *nat.FwNat,
	blendedset blended.Interface,
	informer informerv1.NATInformer,
	commit chan bool) *Controller {
	controller := &Controller{
		cfg:        cfg,
		blendedset: blendedset,
		fwNat:      fwNat,
		lister:     informer.Lister(),
		synced:     informer.Informer().HasSynced,
		queue:      workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "NATs"),
		commit:     commit,
	}
	informer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueue,
		UpdateFunc: func(old, new interface{}) {
			oo := old.(*blendedv1.NAT)
			no := new.(*blendedv1.NAT)
			k8sutil.MakeNeedToUpdate(&no.ObjectMeta, oo.Spec, no.Spec)
			controller.enqueue(no)
		},
	})
	return controller
}

// Run serves the nat controller
func (c *Controller) Run(ctx context.Context, threadiness int) error {
	glog.Info("Starting the nat controller")
	glog.Info("Waiting for the nat informer caches to sync")
	if ok := cache.WaitForCacheSync(ctx.Done(), c.synced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, ctx.Done())
	}
	return nil
}

// Stop stops the nat controller
func (c *Controller) Stop() {
	glog.Info("Stopping the nat controller")
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
			utilruntime.HandleError(fmt.Errorf("NAT expected string in workqueue but got %#v", obj))
			return nil
		}

		if err := c.reconcile(key); err != nil {
			c.queue.AddRateLimited(key)
			return fmt.Errorf("NAT error syncing '%s': %s, requeuing", key, err.Error())
		}

		c.queue.Forget(obj)
		glog.Infof("NAT successfully synced '%s'", key)
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

	nat, err := c.lister.NATs(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("nat '%s' in work queue no longer exists", key))
			return err
		}
		return err
	}

	if !nat.ObjectMeta.DeletionTimestamp.IsZero() {
		if err := c.cleanup(nat); err != nil {
			return err
		}
		return nil
	}

	need := k8sutil.IsNeedToUpdate(nat.ObjectMeta)
	if nat.Status.Phase != blendedv1.NATActive || need {
		if nat.Status.Phase == blendedv1.NATFailed {
			t := util.SubtractNowTime(nat.Status.LastUpdateTime.Time)
			if t.Seconds() <= float64(c.cfg.SyncSec) && !need {
				return nil
			}
		}
		if err := c.createOrUpdate(nat); err != nil {
			return c.makeFailed(nat, err)
		}
	}
	return nil
}

func (c *Controller) makeFailed(nat *blendedv1.NAT, e error) error {
	natCopy := nat.DeepCopy()
	natCopy.Status.Reason = e.Error()
	natCopy.Status.Phase = blendedv1.NATFailed
	natCopy.Status.LastUpdateTime = metav1.NewTime(time.Now())
	delete(natCopy.Annotations, constants.NeedUpdateKey)
	if _, err := c.blendedset.InwinstackV1().NATs(natCopy.Namespace).Update(natCopy); err != nil {
		return err
	}
	glog.Errorf("NAT got an error:%+v.", e)
	return nil
}

func (c *Controller) createOrUpdate(nat *blendedv1.NAT) error {
	natCopy := nat.DeepCopy()
	if err := c.updateNatPolicy(natCopy); err != nil {
		return err
	}

	natCopy.Status.Reason = ""
	natCopy.Status.Phase = blendedv1.NATActive
	natCopy.Status.LastUpdateTime = metav1.NewTime(time.Now())
	delete(natCopy.Annotations, constants.NeedUpdateKey)
	k8sutil.AddFinalizer(&natCopy.ObjectMeta, constants.CustomFinalizer)
	if _, err := c.blendedset.InwinstackV1().NATs(natCopy.Namespace).Update(natCopy); err != nil {
		return err
	}
	return nil
}

func (c *Controller) cleanup(nat *blendedv1.NAT) error {
	natCopy := nat.DeepCopy()
	if err := c.deleteNatPolicy(natCopy); err != nil {
		return err
	}

	k8sutil.RemoveFinalizer(&natCopy.ObjectMeta, constants.CustomFinalizer)
	if _, err := c.blendedset.InwinstackV1().NATs(natCopy.Namespace).Update(natCopy); err != nil {
		return err
	}
	return nil
}
