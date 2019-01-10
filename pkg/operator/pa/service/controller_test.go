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

	"github.com/PaloAltoNetworks/pango/objs/srvc"
	"github.com/PaloAltoNetworks/pango/testdata"
)

func commitSignal(t *testing.T, commit chan bool) {
	for {
		select {
		case <-commit:
			t.Log("Received commit job signal...")
		}
	}
}

func TestServiceController(t *testing.T) {
	client := fake.NewSimpleClientset()
	coreClient := corefake.NewSimpleClientset()
	extensionsClient := extensionsfake.NewSimpleClientset()

	conf := &config.OperatorConfig{
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
	fwSrvc := &srvc.FwSrvc{}
	fwSrvc.Initialize(mc)
	mc.Reset()

	controller := NewController(ctx, client, fwSrvc, conf, ch)
	assert.NotNil(t, controller)

	go commitSignal(t, ch)

	// Test onAdd
	svc := &inwinv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: "k8s-tcp",
		},
		Spec: inwinv1.ServiceSpec{
			Protocol:        "tcp",
			SourcePort:      "",
			DestinationPort: "80,8080",
			Description:     "Test service",
		},
	}

	mc.AddResp("")
	createSvc, err := client.InwinstackV1().Services().Create(svc)
	assert.Nil(t, err)

	controller.onAdd(createSvc)

	onAddSvc, err := client.InwinstackV1().Services().Get(svc.Name, metav1.GetOptions{})
	assert.Nil(t, err)
	assert.NotNil(t, onAddSvc.Status.Phase, inwinv1.ServiceActive)

	mc.AddResp(mc.Elm)
	entry, err := fwSrvc.Get(conf.Vsys, onAddSvc.Name)
	assert.Nil(t, err)
	assert.Equal(t, entry.Name, onAddSvc.Name)
	assert.Equal(t, entry.SourcePort, onAddSvc.Spec.SourcePort)
	assert.Equal(t, entry.DestinationPort, onAddSvc.Spec.DestinationPort)
	assert.Equal(t, entry.Protocol, onAddSvc.Spec.Protocol)

	// Test onUpdate
	mc.AddResp("")
	onAddSvc.Spec.DestinationPort = "9999"
	controller.onUpdate(createSvc, onAddSvc)

	onUpdateSvc, err := client.InwinstackV1().Services().Get(onAddSvc.Name, metav1.GetOptions{})
	assert.Nil(t, err)

	mc.AddResp(mc.Elm)
	onUpdateEntry, err := fwSrvc.Get(conf.Vsys, onUpdateSvc.Name)
	assert.Nil(t, err)
	assert.NotNil(t, onUpdateSvc.Spec.DestinationPort, onUpdateEntry.DestinationPort)

	// Test onDelete
	mc.AddResp("")
	controller.onDelete(onUpdateSvc)
}
