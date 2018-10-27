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
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/golang/glog"
	inwinclientset "github.com/inwinstack/ipam/client/clientset/versioned/typed/inwinstack/v1"
	opkit "github.com/inwinstack/operator-kit"
	"github.com/inwinstack/pa-operator/pkg/constants"
	"github.com/inwinstack/pa-operator/pkg/util"
	"github.com/inwinstack/pa-operator/pkg/util/k8sutil"
	"github.com/inwinstack/pa-operator/pkg/util/pautil"
	"github.com/inwinstack/pa-operator/pkg/util/slice"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

var Resource = opkit.CustomResource{
	Name:    "service",
	Plural:  "services",
	Version: "v1",
	Kind:    reflect.TypeOf(v1.Service{}).Name(),
}

var (
	waitTimeout  = 5 * time.Second
	retryTimeout = 1 * time.Second
	attempts     = 3
)

type ServiceController struct {
	ctx              *opkit.Context
	inwinclient      inwinclientset.InwinstackV1Interface
	pa               *pautil.PaloAlto
	ignoreNamespaces []string
}

func NewController(
	ctx *opkit.Context,
	client inwinclientset.InwinstackV1Interface,
	pa *pautil.PaloAlto,
	namespaces []string) *ServiceController {
	return &ServiceController{
		ctx:              ctx,
		inwinclient:      client,
		pa:               pa,
		ignoreNamespaces: namespaces,
	}
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
	svc := obj.(*v1.Service).DeepCopy()
	glog.V(2).Infof("Received add on Service %s in %s namespace.", svc.Name, svc.Namespace)

	c.makeAnnotations(svc)
	if err := c.createAndUpdate(svc); err != nil {
		glog.Errorf("Failed to create and update on Service %s in %s namespace: %s.", svc.Name, svc.Namespace, err)
	}
}

func (c *ServiceController) onUpdate(oldObj, newObj interface{}) {
	svc := newObj.(*v1.Service).DeepCopy()
	glog.V(2).Infof("Received update on Service %s in %s namespace.", svc.Name, svc.Namespace)

	if svc.DeletionTimestamp == nil {
		if err := c.createAndUpdate(svc); err != nil {
			glog.Errorf("Failed to create and update on Service %s in %s namespace: %s.", svc.Name, svc.Namespace, err)
		}
	}
}

func (c *ServiceController) onDelete(obj interface{}) {
	svc := obj.(*v1.Service).DeepCopy()
	glog.V(2).Infof("Received delete on Service %s in %s namespace.", svc.Name, svc.Namespace)

	if err := c.delete(svc); err != nil {
		glog.Errorf("Failed to delete IP and policies on Service %s in %s namespace: %s.", svc.Name, svc.Namespace, err)
	}
}

func (c *ServiceController) makeAnnotations(svc *v1.Service) {
	if svc.Annotations == nil {
		svc.Annotations = map[string]string{}
	}

	if _, ok := svc.Annotations[constants.AnnKeyAllowSecurity]; !ok {
		svc.Annotations[constants.AnnKeyAllowSecurity] = "false"
	}

	if _, ok := svc.Annotations[constants.AnnKeyAllowNat]; !ok {
		svc.Annotations[constants.AnnKeyAllowNat] = "false"
	}

	if _, ok := svc.Annotations[constants.AnnKeyExternalPool]; !ok {
		svc.Annotations[constants.AnnKeyExternalPool] = constants.DefaultInternetPool
	}
}

func (c *ServiceController) makeRefresh(svc *v1.Service) {
	if ip, ok := svc.Annotations[constants.AnnKeyPublicIP]; ok {
		if util.ParseIP(ip) == nil {
			svc.Annotations[constants.AnnKeyServiceRefresh] = "true"
		}
	}
}

func (c *ServiceController) createAndUpdate(svc *v1.Service) error {
	if slice.Contains(c.ignoreNamespaces, svc.Namespace) {
		return nil
	}

	if len(svc.Spec.Ports) == 0 || len(svc.Spec.ExternalIPs) == 0 || svc.Spec.Type != v1.ServiceTypeLoadBalancer {
		return nil
	}

	if err := c.allocatePublicIP(svc); err != nil {
		return err
	}

	if err := c.createOrDeleteNat(svc); err != nil {
		return err
	}

	if err := c.createOrDeleteSecurity(svc); err != nil {
		return err
	}

	// commit change to PA
	c.pa.Commit()

	c.makeRefresh(svc)
	if _, err := c.ctx.Clientset.CoreV1().Services(svc.Namespace).Update(svc); err != nil {
		return err
	}
	return nil
}

func (c *ServiceController) delete(svc *v1.Service) error {
	if slice.Contains(c.ignoreNamespaces, svc.Namespace) {
		return nil
	}

	if len(svc.Spec.Ports) == 0 || len(svc.Spec.ExternalIPs) == 0 || svc.Spec.Type != v1.ServiceTypeLoadBalancer {
		return nil
	}

	svc.Annotations[constants.AnnKeyAllowNat] = "false"
	svc.Annotations[constants.AnnKeyAllowSecurity] = "false"
	if err := c.createOrDeleteNat(svc); err != nil {
		return err
	}

	if err := c.createOrDeleteSecurity(svc); err != nil {
		return err
	}

	// commit change to PA
	c.pa.Commit()

	if err := c.deallocatePublicIP(svc); err != nil {
		return err
	}
	return nil
}

func (c *ServiceController) allocatePublicIP(svc *v1.Service) error {
	pool := svc.Annotations[constants.AnnKeyExternalPool]
	public := util.ParseIP(svc.Annotations[constants.AnnKeyPublicIP])
	if public == nil && pool != "" {
		ips, err := c.inwinclient.IPs(svc.Namespace).List(metav1.ListOptions{})
		if err != nil {
			return err
		}

		k8sutil.FilterIPs(ips, svc.Spec.ExternalIPs[0], pool)
		if len(ips.Items) != 0 {
			delete(svc.Annotations, constants.AnnKeyServiceRefresh)
			svc.Annotations[constants.AnnKeyPublicIP] = ips.Items[0].Status.Address
			svc.Annotations[constants.AnnKeyPublicID] = ips.Items[0].Name
			return nil
		}

		ip := k8sutil.NewIP(svc.Namespace, pool)
		ip.Annotations = map[string]string{
			constants.AnnKeyExternalIP: svc.Spec.ExternalIPs[0],
		}

		if _, err := c.inwinclient.IPs(svc.Namespace).Create(ip); err != nil {
			return err
		}
	}
	return nil
}

func (c *ServiceController) deallocatePublicIP(svc *v1.Service) error {
	if slice.Contains(c.ignoreNamespaces, svc.Namespace) {
		return nil
	}

	pool := svc.Annotations[constants.AnnKeyExternalPool]
	public := util.ParseIP(svc.Annotations[constants.AnnKeyPublicIP])
	if public != nil && pool != "" {
		svcs, err := c.ctx.Clientset.CoreV1().Services(svc.Namespace).List(metav1.ListOptions{})
		if err != nil {
			return err
		}

		k8sutil.FilterServices(svcs, public.String())
		if len(svcs.Items) != 0 {
			return nil
		}

		id := svc.Annotations[constants.AnnKeyPublicID]
		return c.inwinclient.IPs(svc.Namespace).Delete(id, nil)
	}
	return nil
}

func (c *ServiceController) createOrDeleteNat(svc *v1.Service) error {
	ip := svc.Annotations[constants.AnnKeyPublicIP]
	if util.ParseIP(ip) == nil {
		return nil
	}

	nat := svc.Annotations[constants.AnnKeyAllowNat]
	for _, port := range svc.Spec.Ports {
		proto := strings.ToLower(string(port.Protocol))
		name := fmt.Sprintf("%s-%s-%s-%d", svc.Namespace, svc.Name, ip, port.Port)
		switch nat == "true" {
		case true:
			if err := c.pa.Service.Set(proto, port.Port); err != nil {
				return err
			}

			if err := c.pa.Nat.Set(name, ip, svc.Spec.ExternalIPs[0], port.Port); err != nil {
				return err
			}
		case false:
			c.pa.Nat.Delete(name)
		}
	}
	return nil
}

func (c *ServiceController) createOrDeleteSecurity(svc *v1.Service) error {
	ip := svc.Annotations[constants.AnnKeyPublicIP]
	if util.ParseIP(ip) == nil {
		return nil
	}

	sec := svc.Annotations[constants.AnnKeyAllowSecurity]
	name := fmt.Sprintf("%s-%s-%s", svc.Namespace, svc.Name, ip)
	var services []string
	for _, port := range svc.Spec.Ports {
		proto := strings.ToLower(string(port.Protocol))
		service := fmt.Sprintf("k8s-%s%d", proto, port.Port)
		services = append(services, service)
	}

	switch sec == "true" {
	case true:
		if err := c.pa.Security.Set(name, ip, services); err != nil {
			return err
		}
	case false:
		c.pa.Security.Delete(name)
	}
	return nil
}
