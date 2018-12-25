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
	"github.com/PaloAltoNetworks/pango/objs"
	"github.com/PaloAltoNetworks/pango/objs/srvc"
)

type Service interface {
	List() ([]string, error)
	Get(string) (*srvc.Entry, error)
	Set(*srvc.Entry) error
	Delete(string) error
}

type ServiceOp struct {
	objs *objs.FwObjs
}

var _ Service = &ServiceOp{}

func (op *ServiceOp) List() ([]string, error) {
	svcs, err := op.objs.Services.GetList("")
	if err != nil {
		return nil, err
	}
	return svcs, nil
}

func (op *ServiceOp) Get(name string) (*srvc.Entry, error) {
	svc, err := op.objs.Services.Get("", name)
	if err != nil {
		return nil, err
	}
	return &svc, nil
}

func (op *ServiceOp) Set(entry *srvc.Entry) error {
	return op.objs.Services.Edit("", *entry)
}

func (op *ServiceOp) Delete(name string) error {
	return op.objs.Services.Delete("", name)
}
