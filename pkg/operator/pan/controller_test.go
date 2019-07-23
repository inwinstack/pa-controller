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

package pan

import (
	"context"
	"testing"

	blendedfake "github.com/inwinstack/blended/generated/clientset/versioned/fake"
	blendedinformers "github.com/inwinstack/blended/generated/informers/externalversions"
	"github.com/inwinstack/pa-controller/pkg/config"
	"github.com/inwinstack/pango"
	"github.com/inwinstack/pango/objs"
	"github.com/inwinstack/pango/objs/srvc"
	"github.com/inwinstack/pango/poli"
	"github.com/inwinstack/pango/poli/nat"
	"github.com/inwinstack/pango/poli/security"
	"github.com/stretchr/testify/assert"
)

func commitSignal(t *testing.T, commit chan bool) {
	for {
		select {
		case c := <-commit:
			assert.Equal(t, false, c)
			break
		}
	}
}

func TestPANController(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	fw := &pango.Firewall{
		Policies: &poli.FwPoli{
			Nat:      &nat.FwNat{},
			Security: &security.FwSecurity{},
		},
		Objects: &objs.FwObjs{
			Services: &srvc.FwSrvc{},
		},
	}
	cfg := &config.Config{Threads: 2, Retry: 5}
	blendedset := blendedfake.NewSimpleClientset()
	informer := blendedinformers.NewSharedInformerFactory(blendedset, 0)
	controller := NewController(cfg, fw, blendedset, informer)
	go informer.Start(ctx.Done())
	assert.NotNil(t, controller)
	assert.Nil(t, controller.Run(ctx, cfg.Threads))

	go commitSignal(t, controller.commit)
	controller.commit <- false

	cancel()
	controller.Stop()
}
