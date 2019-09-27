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

package ha

import (
	"context"
	"testing"

	"github.com/inwinstack/pango/testdata"
	"github.com/inwinstack/pango/util"
	"github.com/stretchr/testify/assert"
)

func TestHAInspector(t *testing.T) {
	ch := make(chan bool, 1)
	ctx, cancel := context.WithCancel(context.Background())
	mc := &testdata.MockClient{}

	callbacks := &Callbacks{
		OnActive: func(status *util.HighAvailability) {
			ch <- true
		},
		OnPassive: func() {
			ch <- false
		},
		OnFail: func(err error) {
			ch <- false
			t.Log(err)
		},
	}

	inspector := NewInspector(mc, 30, callbacks)
	assert.Nil(t, inspector.Run(ctx))

	state := <-ch
	assert.Equal(t, true, state)
	cancel()
}
