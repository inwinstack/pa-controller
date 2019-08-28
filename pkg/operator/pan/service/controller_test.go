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

package service

import (
	"context"
	"testing"
	"time"

	blendedv1 "github.com/inwinstack/blended/apis/inwinstack/v1"
	"github.com/inwinstack/blended/constants"
	blendedfake "github.com/inwinstack/blended/generated/clientset/versioned/fake"
	blendedinformers "github.com/inwinstack/blended/generated/informers/externalversions"
	"github.com/inwinstack/pa-controller/pkg/config"
	"github.com/inwinstack/pango/objs/srvc"
	"github.com/inwinstack/pango/testdata"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/stretchr/testify/assert"
)

const timeout = 3 * time.Second

func commitSignal(t *testing.T, commit chan bool, stopCh <-chan struct{}) {
	for {
		select {
		case c := <-commit:
			assert.Equal(t, true, c)
		case <-stopCh:
			return
		}
	}
}

func TestServiceController(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	commit := make(chan bool, 1)
	cfg := &config.Config{Threads: 2, Retry: 5}
	blendedset := blendedfake.NewSimpleClientset()
	informer := blendedinformers.NewSharedInformerFactory(blendedset, 0)

	// PAN firewall fake client
	mc := &testdata.MockClient{}
	fwSrvc := &srvc.FwSrvc{}
	fwSrvc.Initialize(mc)

	controller := NewController(cfg, fwSrvc, blendedset, informer.Inwinstack().V1().Services(), commit)
	go informer.Start(ctx.Done())
	go commitSignal(t, controller.commit, ctx.Done())
	assert.Nil(t, controller.Run(ctx, cfg.Threads))

	svc := &blendedv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: "k8s-tcp",
		},
		Spec: blendedv1.ServiceSpec{
			Protocol:        "tcp",
			SourcePort:      "",
			DestinationPort: "80,8080",
			Description:     "Test service",
		},
	}

	mc.Reset()
	mc.AddResp("")
	_, err := blendedset.InwinstackV1().Services().Create(svc)
	assert.Nil(t, err)

	failed := true
	for start := time.Now(); time.Since(start) < timeout; {
		gsvc, err := blendedset.InwinstackV1().Services().Get(svc.Name, metav1.GetOptions{})
		assert.Nil(t, err)

		mc.AddResp(mc.Elm)
		entry, err := fwSrvc.Get(cfg.Vsys, gsvc.Name)
		assert.Nil(t, err)
		if gsvc.Status.Phase == blendedv1.ServiceActive && entry.Name != "" {
			assert.Equal(t, []string{constants.CustomFinalizer}, gsvc.Finalizers)
			assert.Equal(t, gsvc.Name, entry.Name)
			assert.Equal(t, gsvc.Spec.SourcePort, entry.SourcePort)
			assert.Equal(t, gsvc.Spec.DestinationPort, entry.DestinationPort)
			assert.Equal(t, gsvc.Spec.Protocol, entry.Protocol)
			failed = false
			break
		}
	}
	assert.Equal(t, false, failed, "The service object hasn't created.")
	assert.Nil(t, blendedset.InwinstackV1().Services().Delete(svc.Name, nil))
	svcList, err := blendedset.InwinstackV1().Services().List(metav1.ListOptions{})
	assert.Nil(t, err)
	assert.Equal(t, 0, len(svcList.Items))

	// TODO(k2r2bai): The mock client hasn't implement deleting.

	cancel()
	mc.Reset()
	controller.Stop()
}
