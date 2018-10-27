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
	"github.com/PaloAltoNetworks/pango"
)

type PaloAltoFlag struct {
	Host     string
	Username string
	Password string
}

type PaloAlto struct {
	client   *pango.Firewall
	Nat      Nat
	Security Security
	Service  Service
}

func NewClient(flag *PaloAltoFlag) (*PaloAlto, error) {
	client := &pango.Firewall{Client: pango.Client{
		Hostname: flag.Host,
		Username: flag.Username,
		Password: flag.Password,
		Logging:  pango.LogQuiet,
	}}

	if err := client.Initialize(); err != nil {
		return nil, err
	}

	pa := &PaloAlto{client: client}
	pa.Nat = &NatOp{policies: client.Policies}
	pa.Security = &SecurityOp{policies: client.Policies}
	pa.Service = &ServiceOp{objs: client.Objects}
	return pa, nil
}

func (pa *PaloAlto) Commit() error {
	_, err := pa.client.Commit("", false, true, false, false)
	if err != nil {
		return err
	}
	return nil
}

func (pa *PaloAlto) GetVersion() string {
	return pa.client.Versioning().String()
}

func (pa *PaloAlto) GetHostname() string {
	return pa.client.Hostname
}

func (pa *PaloAlto) GetUsername() string {
	return pa.client.Username
}
