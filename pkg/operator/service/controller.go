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

	"github.com/golang/glog"
	inwinclientset "github.com/inwinstack/blended/client/clientset/versioned/typed/inwinstack/v1"
	opkit "github.com/inwinstack/operator-kit"
	"github.com/inwinstack/pa-operator/pkg/constants"
	"github.com/inwinstack/pa-operator/pkg/k8sutil"
	"github.com/inwinstack/pa-operator/pkg/pautil"
	"github.com/inwinstack/pa-operator/pkg/util"
	"github.com/inwinstack/pa-operator/pkg/util/slice"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/tools/cache"
)

var Resource = opkit.CustomResource{
	Name:    "service",
	Plural:  "services",
	Version: "v1",
	Kind:    reflect.TypeOf(v1.Service{}).Name(),
}

type ServiceController struct {
	ctx              *opkit.Context
	inwinclient      inwinclientset.InwinstackV1Interface
	paclient         *pautil.PaloAlto
	ignoreNamespaces []string
}

func NewController(
	ctx *opkit.Context,
	client inwinclientset.InwinstackV1Interface,
	paclient *pautil.PaloAlto,
	namespaces []string) *ServiceController {
	return &ServiceController{
		ctx:              ctx,
		inwinclient:      client,
		paclient:         paclient,
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
	if err := c.syncSpec(nil, svc); err != nil {
		glog.Errorf("Failed to sync spec on Service %s in %s namespace: %s.", svc.Name, svc.Namespace, err)
	}
}

func (c *ServiceController) onUpdate(oldObj, newObj interface{}) {
	old := oldObj.(*v1.Service).DeepCopy()
	svc := newObj.(*v1.Service).DeepCopy()
	glog.V(2).Infof("Received update on Service %s in %s namespace.", svc.Name, svc.Namespace)

	if svc.DeletionTimestamp == nil {
		if err := c.syncSpec(old, svc); err != nil {
			glog.Errorf("Failed to sync spec on Service %s in %s namespace: %s.", svc.Name, svc.Namespace, err)
		}
	}
}

func (c *ServiceController) onDelete(obj interface{}) {
	svc := obj.(*v1.Service).DeepCopy()
	glog.V(2).Infof("Received delete on Service %s in %s namespace.", svc.Name, svc.Namespace)

	if slice.Contains(c.ignoreNamespaces, svc.Namespace) {
		return
	}

	if len(svc.Spec.Ports) == 0 || len(svc.Spec.ExternalIPs) == 0 {
		return
	}

	if err := c.deallocatePublicIP(svc); err != nil {
		glog.Errorf("Failed to deallocate IP on Service %s in %s namespace: %s.", svc.Name, svc.Namespace, err)
	}
}

func (c *ServiceController) makeAnnotations(svc *v1.Service) {
	if svc.Annotations == nil {
		svc.Annotations = map[string]string{}
	}

	if _, ok := svc.Annotations[constants.AnnKeyAllowSecurity]; !ok {
		svc.Annotations[constants.AnnKeyAllowSecurity] = "false"
	}

	if _, ok := svc.Annotations[constants.AnnKeyAllowNAT]; !ok {
		svc.Annotations[constants.AnnKeyAllowNAT] = "false"
	}

	if _, ok := svc.Annotations[constants.AnnKeyExternalPool]; !ok {
		svc.Annotations[constants.AnnKeyExternalPool] = constants.DefaultInternetPool
	}
}

func (c *ServiceController) makeRefresh(svc *v1.Service) {
	ip := svc.Annotations[constants.AnnKeyPublicIP]
	if util.ParseIP(ip) == nil {
		svc.Annotations[constants.AnnKeyServiceRefresh] = string(uuid.NewUUID())
	}
}

func (c *ServiceController) syncSpec(old *v1.Service, svc *v1.Service) error {
	if slice.Contains(c.ignoreNamespaces, svc.Namespace) {
		return nil
	}

	if len(svc.Spec.Ports) == 0 || len(svc.Spec.ExternalIPs) == 0 {
		return nil
	}

	if err := c.allocatePublicIP(svc); err != nil {
		glog.Errorf("Failed to allocate Public IP: %s.", err)
	}

	ip := svc.Annotations[constants.AnnKeyPublicIP]
	if util.ParseIP(ip) != nil {
		ports := k8sutil.MarkChangePorts(old, svc)
		if err := c.syncNAT(svc, ip, ports); err != nil {
			glog.Errorf("Failed to create and update NAT: %s.", err)
		}

		if err := c.syncSecurity(svc, ip, ports); err != nil {
			glog.Errorf("Failed to create and update Security: %s.", err)
		}
	}

	c.makeRefresh(svc)
	if _, err := c.ctx.Clientset.CoreV1().Services(svc.Namespace).Update(svc); err != nil {
		return err
	}
	return nil
}

func (c *ServiceController) allocatePublicIP(svc *v1.Service) error {
	pool := svc.Annotations[constants.AnnKeyExternalPool]
	public := util.ParseIP(svc.Annotations[constants.AnnKeyPublicIP])
	if public == nil && pool != "" {
		name := svc.Spec.ExternalIPs[0]
		ip, err := c.inwinclient.IPs(svc.Namespace).Get(name, metav1.GetOptions{})
		if err == nil {
			if ip.Status.Address != "" {
				delete(svc.Annotations, constants.AnnKeyServiceRefresh)
				svc.Annotations[constants.AnnKeyPublicIP] = ip.Status.Address
			}
			return nil
		}

		newIP := k8sutil.NewIP(svc.Spec.ExternalIPs[0], svc.Namespace, pool)
		if _, err := c.inwinclient.IPs(svc.Namespace).Create(newIP); err != nil {
			return err
		}
	}
	return nil
}

func (c *ServiceController) deallocatePublicIP(svc *v1.Service) error {
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
		return c.inwinclient.IPs(svc.Namespace).Delete(svc.Spec.ExternalIPs[0], nil)
	}
	return nil
}

func (c *ServiceController) syncNAT(svc *v1.Service, ip string, ports map[v1.ServicePort]bool) error {
	t := svc.Annotations[constants.AnnKeyAllowNAT]
	for port, retain := range ports {
		proto := strings.ToLower(string(port.Protocol))
		name := fmt.Sprintf("%s-%d", ip, port.Port)
		switch {
		case t == "true" && retain:
			if err := c.paclient.Service.Set(proto, port.Port); err != nil {
				return err
			}

			if err := k8sutil.CreateOrUpdateNAT(c.inwinclient, name, ip, port.Port, svc); err != nil {
				return err
			}
		default:
			c.inwinclient.NATs(svc.Namespace).Delete(name, nil)
		}
	}
	return nil
}

func (c *ServiceController) syncSecurity(svc *v1.Service, ip string, ports map[v1.ServicePort]bool) error {
	t := svc.Annotations[constants.AnnKeyAllowSecurity]
	for port, retain := range ports {
		proto := strings.ToLower(string(port.Protocol))
		service := fmt.Sprintf("k8s-%s%d", proto, port.Port)
		name := fmt.Sprintf("%s-%d", ip, port.Port)
		switch {
		case t == "true" && retain:
			if err := k8sutil.CreateOrUpdateSecurity(c.inwinclient, name, ip, service, svc); err != nil {
				return err
			}
		default:
			c.inwinclient.Securities(svc.Namespace).Delete(ip, nil)
		}
	}
	return nil
}
