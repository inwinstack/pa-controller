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

package nat

import (
	"testing"
	"time"

	inwinv1 "github.com/inwinstack/blended/apis/inwinstack/v1"
	fake "github.com/inwinstack/blended/client/clientset/versioned/fake"
	opkit "github.com/inwinstack/operator-kit"

	"github.com/inwinstack/pa-controller/pkg/config"
	extensionsfake "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	corefake "k8s.io/client-go/kubernetes/fake"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/PaloAltoNetworks/pango/poli/nat"
	"github.com/PaloAltoNetworks/pango/testdata"
)

const namespace = "default"

func commitSignal(t *testing.T, commit chan bool) {
	for {
		select {
		case <-commit:
			t.Log("Received commit job signal...")
		}
	}
}

func TestNATController(t *testing.T) {
	client := fake.NewSimpleClientset()
	coreClient := corefake.NewSimpleClientset()
	extensionsClient := extensionsfake.NewSimpleClientset()

	conf := &config.Operator{
		Host:     "172.22.132.200",
		Username: "admin",
		Password: "admin",
		Vsys:     "",
	}

	ctx := &opkit.Context{
		Clientset:             coreClient,
		APIExtensionClientset: extensionsClient,
		Interval:              500 * time.Millisecond,
		Timeout:               60 * time.Second,
	}

	ch := make(chan bool)
	mc := &testdata.MockClient{}
	fwNat := &nat.FwNat{}
	fwNat.Initialize(mc)
	mc.Reset()

	controller := NewController(ctx, client, fwNat, conf, ch)
	assert.NotNil(t, controller)

	go commitSignal(t, ch)

	// Test onAdd
	nat := &inwinv1.NAT{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-nat",
		},
		Spec: inwinv1.NATSpec{
			Type:                 inwinv1.NATIPv4,
			SourceZones:          []string{"untrust"},
			SourceAddresses:      []string{"any"},
			DestinationAddresses: []string{"140.23.110.10"},
			DestinationZone:      "untrust",
			DatType:              inwinv1.NATDatStatic,
			DatAddress:           "172.22.132.10",
		},
	}

	mc.AddResp("")
	createNat, err := client.InwinstackV1().NATs(namespace).Create(nat)
	assert.Nil(t, err)

	controller.onAdd(createNat)

	onAddNat, err := client.InwinstackV1().NATs(namespace).Get(nat.Name, metav1.GetOptions{})
	assert.Nil(t, err)
	assert.Equal(t, inwinv1.NATActive, onAddNat.Status.Phase)

	mc.AddResp(mc.Elm)
	entry, err := fwNat.Get(conf.Vsys, onAddNat.Name)
	assert.Nil(t, err)
	assert.Equal(t, onAddNat.Name, entry.Name)
	assert.Equal(t, onAddNat.Spec.Type, entry.Type)
	assert.Equal(t, onAddNat.Spec.SourceZones, entry.SourceZones)
	assert.Equal(t, onAddNat.Spec.SourceAddresses, entry.SourceAddresses)
	assert.Equal(t, onAddNat.Spec.DestinationAddresses, entry.DestinationAddresses)
	assert.Equal(t, onAddNat.Spec.DestinationZone, entry.DestinationZone)
	assert.Equal(t, onAddNat.Spec.DatType, entry.DatType)
	assert.Equal(t, onAddNat.Spec.DatAddress, entry.DatAddress)

	// Test onUpdate
	mc.AddResp("")
	onAddNat.Spec.DatAddress = "172.22.132.11"
	onAddNat.Spec.DestinationAddresses = []string{"140.23.110.11"}
	controller.onUpdate(createNat, onAddNat)

	onUpdateNat, err := client.InwinstackV1().NATs(namespace).Get(onAddNat.Name, metav1.GetOptions{})
	assert.Nil(t, err)

	mc.AddResp(mc.Elm)
	onUpdateEntry, err := fwNat.Get(conf.Vsys, onUpdateNat.Name)
	assert.Nil(t, err)
	assert.Equal(t, onUpdateNat.Spec.DatAddress, onUpdateEntry.DatAddress)

	// Test onDelete
	// PA mock hasn’t implement delete API.
	mc.AddResp("")
	controller.onDelete(onUpdateNat)
}
