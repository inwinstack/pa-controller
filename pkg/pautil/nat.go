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
	"github.com/PaloAltoNetworks/pango/poli"
	"github.com/PaloAltoNetworks/pango/poli/nat"
)

type Nat interface {
	List() ([]string, error)
	Get(string) (*nat.Entry, error)
	Set(*nat.Entry) error
	Delete(string) error
}

type NatOp struct {
	policies *poli.FwPoli
}

var _ Nat = &NatOp{}

func (op *NatOp) List() ([]string, error) {
	policies, err := op.policies.Nat.GetList("")
	if err != nil {
		return nil, err
	}
	return policies, nil
}

func (op *NatOp) Get(name string) (*nat.Entry, error) {
	entry, err := op.policies.Nat.Get("", name)
	if err != nil {
		return nil, err
	}
	return &entry, nil
}

func (op *NatOp) Set(entry *nat.Entry) error {
	if err := op.policies.Nat.Edit("", *entry); err != nil {
		return err
	}
	return nil
}

func (op *NatOp) Delete(name string) error {
	if err := op.policies.Nat.Delete("", name); err != nil {
		return err
	}
	return nil
}
