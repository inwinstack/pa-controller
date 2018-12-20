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

func newSecurity(name, srcAddr, service string, svc *v1.Service) *inwinv1.Security {
	return &inwinv1.Security{
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
		Spec: inwinv1.SecuritySpec{
			Description:                     "Auto sync Security for Kubernetes.",
			SourceZones:                     []string{"untrust"},
			SourceAddresses:                 []string{"any"},
			SourceUsers:                     []string{"any"},
			HipProfiles:                     []string{"any"},
			DestinationZones:                []string{"AI public service network"},
			DestinationAddresses:            []string{srcAddr},
			Applications:                    []string{"any"},
			Services:                        []string{service},
			Categories:                      []string{"any"},
			Action:                          "allow",
			IcmpUnreachable:                 false,
			DisableServerResponseInspection: false,
			LogEnd:                          true,
			LogSetting:                      "siem_forward",
			Group:                           "inwin-monitor",
		},
	}
}

func CreateOrUpdateSecurity(c inwinclientset.InwinstackV1Interface, name, srcAddr, service string, svc *v1.Service) error {
	sec, err := c.Securities(svc.Namespace).Get(name, metav1.GetOptions{})
	if err == nil {
		sec.Spec.DestinationAddresses = []string{srcAddr}
		sec.Spec.Services = []string{service}
		if _, err := c.Securities(svc.Namespace).Update(sec); err != nil {
			return err
		}
		return nil
	}

	newSec := newSecurity(name, srcAddr, service, svc)
	if _, err := c.Securities(svc.Namespace).Create(newSec); err != nil {
		return err
	}
	return nil
}
