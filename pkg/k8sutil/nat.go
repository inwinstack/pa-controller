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
	"reflect"

	inwinv1 "github.com/inwinstack/blended/apis/inwinstack/v1"
	inwinclientset "github.com/inwinstack/blended/client/clientset/versioned/typed/inwinstack/v1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func newNAT(name, srcAddr, service string, port int32, svc *v1.Service) *inwinv1.NAT {
	return &inwinv1.NAT{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: svc.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(svc, schema.GroupVersionKind{
					Group:   v1.SchemeGroupVersion.Group,
					Version: v1.SchemeGroupVersion.Version,
					Kind:    reflect.TypeOf(v1.Service{}).Name(),
				}),
			},
		},
		Spec: inwinv1.NATSpec{
			Type:                 inwinv1.NATIPv4,
			Description:          "Auto sync NAT for Kubernetes.",
			SourceZones:          []string{"untrust"},
			SourceAddresses:      []string{"any"},
			DestinationAddresses: []string{srcAddr},
			DestinationZone:      "untrust",
			ToInterface:          "any",
			Service:              service,
			SatType:              inwinv1.NATSatNone,
			DatType:              inwinv1.NATDatStatic,
			DatAddress:           svc.Spec.ExternalIPs[0],
			DatPort:              port,
		},
	}
}

func CreateOrUpdateNAT(c inwinclientset.InwinstackV1Interface, name, srcAddr, service string, port int32, svc *v1.Service) error {
	nat, err := c.NATs(svc.Namespace).Get(name, metav1.GetOptions{})
	if err == nil {
		nat.Spec.DestinationAddresses = []string{srcAddr}
		nat.Spec.DatAddress = svc.Spec.ExternalIPs[0]
		nat.Spec.DatPort = port
		if _, err := c.NATs(svc.Namespace).Update(nat); err != nil {
			return err
		}
		return nil
	}

	newNAT := newNAT(name, srcAddr, service, port, svc)
	if _, err := c.NATs(svc.Namespace).Create(newNAT); err != nil {
		return err
	}
	return nil
}
