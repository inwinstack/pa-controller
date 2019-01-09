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

package pautil

import (
	"testing"

	inwinv1 "github.com/inwinstack/blended/apis/inwinstack/v1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestToSecurityEntry(t *testing.T) {
	sec := &inwinv1.Security{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-sec",
		},
		Spec: inwinv1.SecuritySpec{
			DestinationAddresses: []string{"140.23.110.10"},
			DestinationZones:     []string{"trust"},
			Services:             []string{"k8s-tcp", "k8s-udp"},
			SourceZones:          []string{"untrust"},
		},
	}

	entry := ToSecurityEntry(sec)
	assert.Equal(t, sec.Name, entry.Name)
	assert.Equal(t, sec.Spec.DestinationZones, entry.DestinationZones)
	assert.Equal(t, sec.Spec.DestinationAddresses, entry.DestinationAddresses)
	assert.Equal(t, sec.Spec.SourceZones, entry.SourceZones)
	assert.Equal(t, sec.Spec.Services, entry.Services)
}

func TestToNatEntry(t *testing.T) {
	nat := &inwinv1.NAT{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-nat",
		},
		Spec: inwinv1.NATSpec{
			DestinationAddresses: []string{"140.23.110.10"},
			DatAddress:           "172.22.132.10",
			DatPort:              8080,
			Service:              "any",
			SourceZones:          []string{"untrust"},
			Type:                 inwinv1.NATIPv4,
		},
	}

	entry := ToNatEntry(nat)
	assert.Equal(t, nat.Name, entry.Name)
	assert.Equal(t, nat.Spec.DestinationAddresses, entry.DestinationAddresses)
	assert.Equal(t, nat.Spec.DatAddress, entry.DatAddress)
	assert.Equal(t, nat.Spec.DatPort, int32(entry.DatPort))
	assert.Equal(t, nat.Spec.Service, entry.Service)
	assert.Equal(t, nat.Spec.SourceZones, entry.SourceZones)
	assert.Equal(t, nat.Spec.Type, entry.Type)
}

func TestToServiceEntry(t *testing.T) {
	srvc := &inwinv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-srvc",
		},
		Spec: inwinv1.ServiceSpec{
			DestinationPort: "80",
			Protocol:        "tcp",
		},
	}

	entry := ToServiceEntry(srvc)
	assert.Equal(t, srvc.Name, entry.Name)
	assert.Equal(t, srvc.Spec.DestinationPort, entry.DestinationPort)
	assert.Equal(t, srvc.Spec.Protocol, entry.Protocol)
}
