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

package pa

import (
	"testing"

	fake "github.com/inwinstack/blended/client/clientset/versioned/fake"
	opkit "github.com/inwinstack/operator-kit"

	"github.com/inwinstack/pa-controller/pkg/config"

	"github.com/stretchr/testify/assert"
)

const namespace = "default"

func TestController(t *testing.T) {
	client := fake.NewSimpleClientset()
	pwdConf := &config.Operator{
		Host:     "172.22.132.200",
		Username: "admin",
		Password: "admin",
		Vsys:     "",
	}

	keyConf := &config.Operator{
		Host:     "172.22.132.200",
		Username: "admin",
		APIKey:   "SSdtIFNoYW1hbiBLaW5nCg==",
		Vsys:     "",
	}

	ctx := &opkit.Context{}

	pwdCtrl := NewController(ctx, client, pwdConf)
	assert.NotNil(t, pwdCtrl)
	assert.Equal(t, pwdConf.Password, pwdCtrl.conf.Password)
	assert.Equal(t, "", pwdCtrl.conf.APIKey)

	keyCtrl := NewController(ctx, client, keyConf)
	assert.NotNil(t, keyCtrl)
	assert.Equal(t, keyConf.APIKey, keyCtrl.conf.APIKey)
	assert.Equal(t, "", keyCtrl.conf.Password)
}
