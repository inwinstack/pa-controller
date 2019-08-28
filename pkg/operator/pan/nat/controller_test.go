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

package nat

import (
	"context"
	"testing"
	"time"

	blendedv1 "github.com/inwinstack/blended/apis/inwinstack/v1"
	"github.com/inwinstack/blended/constants"
	blendedfake "github.com/inwinstack/blended/generated/clientset/versioned/fake"
	blendedinformers "github.com/inwinstack/blended/generated/informers/externalversions"
	"github.com/inwinstack/pa-controller/pkg/config"
	"github.com/inwinstack/pango/poli/nat"
	"github.com/inwinstack/pango/testdata"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func TestNATController(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	commit := make(chan bool, 1)
	cfg := &config.Config{Threads: 2, Retry: 5}
	blendedset := blendedfake.NewSimpleClientset()
	informer := blendedinformers.NewSharedInformerFactory(blendedset, 0)

	// PAN firewall fake client
	mc := &testdata.MockClient{}
	fwNat := &nat.FwNat{}
	fwNat.Initialize(mc)

	controller := NewController(cfg, fwNat, blendedset, informer.Inwinstack().V1().NATs(), commit)
	go informer.Start(ctx.Done())
	go commitSignal(t, controller.commit, ctx.Done())
	assert.Nil(t, controller.Run(ctx, cfg.Threads))

	namespace := "default"
	nat := &blendedv1.NAT{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-nat",
		},
		Spec: blendedv1.NATSpec{
			Type:                 blendedv1.NATIPv4,
			SourceZones:          []string{"untrust"},
			SourceAddresses:      []string{"any"},
			DestinationAddresses: []string{"140.23.110.10"},
			DestinationZone:      "untrust",
			DatType:              blendedv1.NATDatStatic,
			DatAddress:           "172.22.132.10",
		},
	}

	mc.Reset()
	mc.AddResp("")
	_, err := blendedset.InwinstackV1().NATs(namespace).Create(nat)
	assert.Nil(t, err)

	failed := true
	for start := time.Now(); time.Since(start) < timeout; {
		gnat, err := blendedset.InwinstackV1().NATs(namespace).Get(nat.Name, metav1.GetOptions{})
		assert.Nil(t, err)

		mc.AddResp(mc.Elm)
		entry, err := fwNat.Get(cfg.Vsys, gnat.Name)
		assert.Nil(t, err)
		if gnat.Status.Phase == blendedv1.NATActive && entry.Name != "" {
			assert.Equal(t, []string{constants.CustomFinalizer}, gnat.Finalizers)
			assert.Equal(t, gnat.Name, entry.Name)
			assert.Equal(t, gnat.Spec.Type, entry.Type)
			assert.Equal(t, gnat.Spec.SourceZones, entry.SourceZones)
			assert.Equal(t, gnat.Spec.SourceAddresses, entry.SourceAddresses)
			assert.Equal(t, gnat.Spec.DestinationAddresses, entry.DestinationAddresses)
			assert.Equal(t, gnat.Spec.DestinationZone, entry.DestinationZone)
			assert.Equal(t, gnat.Spec.DatType, entry.DatType)
			assert.Equal(t, gnat.Spec.DatAddress, entry.DatAddress)
			failed = false
			break
		}
	}
	assert.Equal(t, false, failed, "The nat policy hasn't created.")
	assert.Nil(t, blendedset.InwinstackV1().NATs(namespace).Delete(nat.Name, nil))
	natList, err := blendedset.InwinstackV1().NATs(namespace).List(metav1.ListOptions{})
	assert.Nil(t, err)
	assert.Equal(t, 0, len(natList.Items))

	// TODO(k2r2bai): The mock client hasn't implement deleting.

	cancel()
	mc.Reset()
	controller.Stop()
}
