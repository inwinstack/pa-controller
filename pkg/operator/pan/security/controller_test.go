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

package security

import (
	"context"
	"reflect"
	"testing"
	"time"

	blendedv1 "github.com/inwinstack/blended/apis/inwinstack/v1"
	"github.com/inwinstack/blended/constants"
	blendedfake "github.com/inwinstack/blended/generated/clientset/versioned/fake"
	blendedinformers "github.com/inwinstack/blended/generated/informers/externalversions"
	"github.com/inwinstack/pa-controller/pkg/config"
	"github.com/inwinstack/pango/poli/security"
	"github.com/inwinstack/pango/testdata"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/stretchr/testify/assert"
)

const timeout = 2 * time.Second

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

func TestSecurityController(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	commit := make(chan bool, 1)
	cfg := &config.Config{Threads: 2, Retry: 5}
	blendedset := blendedfake.NewSimpleClientset()
	informer := blendedinformers.NewSharedInformerFactory(blendedset, 0)

	// PAN firewall fake client
	mc := &testdata.MockClient{}
	fwSec := &security.FwSecurity{}
	fwSec.Initialize(mc)

	controller := NewController(cfg, fwSec, blendedset, informer.Inwinstack().V1().Securities(), commit)
	go informer.Start(ctx.Done())
	go commitSignal(t, controller.commit, ctx.Done())
	assert.Nil(t, controller.Run(ctx, cfg.Threads))

	namespace := "default"
	sec := &blendedv1.Security{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sec",
			Namespace: namespace,
		},
		Spec: blendedv1.SecuritySpec{
			SourceZones:          []string{"untrust"},
			SourceAddresses:      []string{"any"},
			SourceUsers:          []string{"any"},
			HipProfiles:          []string{"any"},
			DestinationZones:     []string{"AI public service network"},
			DestinationAddresses: []string{"140.23.110.10"},
			Applications:         []string{"any"},
			Categories:           []string{"any"},
			Services:             []string{"k8s-tcp80"},
			Action:               blendedv1.SecurityAllow,
		},
	}

	mc.Reset()
	mc.AddResp("")
	_, err := blendedset.InwinstackV1().Securities(namespace).Create(sec)
	assert.Nil(t, err)

	failed := true
	for start := time.Now(); time.Since(start) < timeout; {
		gsec, err := blendedset.InwinstackV1().Securities(namespace).Get(sec.Name, metav1.GetOptions{})
		assert.Nil(t, err)

		mc.AddResp(mc.Elm)
		entry, err := fwSec.Get(cfg.Vsys, gsec.Name)
		assert.Nil(t, err)
		if gsec.Status.Phase == blendedv1.SecurityActive && entry.Name != "" {
			assert.Equal(t, []string{constants.CustomFinalizer}, gsec.Finalizers)
			assert.Equal(t, gsec.Name, entry.Name)
			assert.Equal(t, gsec.Spec.Services, entry.Services)
			assert.Equal(t, gsec.Spec.DestinationAddresses, entry.DestinationAddresses)
			assert.Equal(t, gsec.Spec.DestinationZones, entry.DestinationZones)
			assert.Equal(t, gsec.Spec.Action, entry.Action)
			failed = false
			break
		}
	}
	assert.Equal(t, false, failed, "The security policy hasn't created.")

	gsec2, err := blendedset.InwinstackV1().Securities(namespace).Get(sec.Name, metav1.GetOptions{})
	assert.Nil(t, err)

	mc.Reset()
	mc.AddResp("")
	gsec2.Spec.DestinationAddresses = []string{"140.23.110.12"}
	usec, err := blendedset.InwinstackV1().Securities(namespace).Update(gsec2)
	assert.Nil(t, err)

	failed = true
	for start := time.Now(); time.Since(start) < timeout; {
		mc.AddResp(mc.Elm)
		enrty, err := fwSec.Get(cfg.Vsys, usec.Name)
		assert.Nil(t, err)
		if reflect.DeepEqual(usec.Spec.DestinationAddresses, enrty.DestinationAddresses) {
			failed = false
			break
		}
	}
	assert.Equal(t, false, failed, "The security policy hasn't synced.")

	assert.Nil(t, blendedset.InwinstackV1().Securities(namespace).Delete(sec.Name, nil))
	secList, err := blendedset.InwinstackV1().Securities(namespace).List(metav1.ListOptions{})
	assert.Nil(t, err)
	assert.Equal(t, 0, len(secList.Items))

	// TODO(k2r2bai): The mock client hasn't implement deleting.

	cancel()
	mc.Reset()
	controller.Stop()
}
