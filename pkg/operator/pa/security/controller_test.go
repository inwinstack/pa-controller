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

package security

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

	"github.com/PaloAltoNetworks/pango/poli/security"
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

func TestSecurityController(t *testing.T) {
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
	fwSec := &security.FwSecurity{}
	fwSec.Initialize(mc)
	mc.Reset()

	controller := NewController(ctx, client, fwSec, conf, ch)
	assert.NotNil(t, controller)

	go commitSignal(t, ch)

	// Test onAdd
	sec := &inwinv1.Security{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-sec",
		},
		Spec: inwinv1.SecuritySpec{
			SourceZones:          []string{"untrust"},
			SourceAddresses:      []string{"any"},
			SourceUsers:          []string{"any"},
			HipProfiles:          []string{"any"},
			DestinationZones:     []string{"AI public service network"},
			DestinationAddresses: []string{"140.23.110.10"},
			Applications:         []string{"any"},
			Categories:           []string{"any"},
			Services:             []string{"k8s-tcp80"},
			Action:               inwinv1.SecurityAllow,
		},
	}

	mc.AddResp("")
	createSec, err := client.InwinstackV1().Securities(namespace).Create(sec)
	assert.Nil(t, err)

	controller.onAdd(createSec)

	onAddSec, err := client.InwinstackV1().Securities(namespace).Get(sec.Name, metav1.GetOptions{})
	assert.Nil(t, err)
	assert.Equal(t, inwinv1.SecurityActive, onAddSec.Status.Phase)

	mc.AddResp(mc.Elm)
	entry, err := fwSec.Get(conf.Vsys, onAddSec.Name)
	assert.Nil(t, err)
	assert.Equal(t, onAddSec.Name, entry.Name)
	assert.Equal(t, onAddSec.Spec.Services, entry.Services)
	assert.Equal(t, onAddSec.Spec.DestinationAddresses, entry.DestinationAddresses)
	assert.Equal(t, onAddSec.Spec.DestinationZones, entry.DestinationZones)
	assert.Equal(t, onAddSec.Spec.Action, entry.Action)

	// Test onUpdate
	mc.AddResp("")
	onAddSec.Spec.DestinationAddresses = []string{"140.23.110.12"}
	controller.onUpdate(createSec, onAddSec)

	onUpdateSec, err := client.InwinstackV1().Securities(namespace).Get(onAddSec.Name, metav1.GetOptions{})
	assert.Nil(t, err)

	mc.AddResp(mc.Elm)
	onUpdateEntry, err := fwSec.Get(conf.Vsys, onUpdateSec.Name)
	assert.Nil(t, err)
	assert.Equal(t, onUpdateSec.Spec.DestinationAddresses, onUpdateEntry.DestinationAddresses)

	// Test onDelete
	// PA mock hasn’t implement delete API.
	mc.AddResp("")
	controller.onDelete(onUpdateSec)
}
