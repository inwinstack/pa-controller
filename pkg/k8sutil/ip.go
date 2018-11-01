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

package k8sutil

import (
	"time"

	"github.com/golang/glog"
	inwinv1 "github.com/inwinstack/blended/apis/inwinstack/v1"
	inwinclientset "github.com/inwinstack/blended/client/clientset/versioned/typed/inwinstack/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
)

func NewIP(name, namespace, poolName string) *inwinv1.IP {
	return &inwinv1.IP{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: inwinv1.IPSpec{
			PoolName:        poolName,
			UpdateNamespace: false,
		},
	}
}

func WaitForIP(c inwinclientset.InwinstackV1Interface, ns, name string, timeout time.Duration) error {
	opts := metav1.ListOptions{
		FieldSelector: fields.Set{
			"metadata.name":      name,
			"metadata.namespace": ns,
		}.AsSelector().String()}

	w, err := c.IPs(ns).Watch(opts)
	if err != nil {
		return err
	}

	_, err = watch.Until(timeout, w, func(event watch.Event) (bool, error) {
		switch event.Type {
		case watch.Deleted:
			return false, apierrs.NewNotFound(schema.GroupResource{Resource: "ips"}, "")
		}

		switch ip := event.Object.(type) {
		case *inwinv1.IP:
			if ip.Name == name &&
				ip.Namespace == ns &&
				ip.Status.Phase == inwinv1.IPActive {
				return true, nil
			}
			glog.V(2).Infof("Waiting for IP %s to stabilize, generation %v observed status.IP %s.",
				name, ip.Generation, ip.Status.Address)
		}
		return false, nil
	})
	return nil
}
