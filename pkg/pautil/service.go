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
	"fmt"

	"github.com/PaloAltoNetworks/pango/objs"
	"github.com/PaloAltoNetworks/pango/objs/srvc"
)

type Service interface {
	List() ([]string, error)
	Get(string) (*srvc.Entry, error)
	Set(string, int32) error
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

func (op *ServiceOp) Set(protocol string, port int32) error {
	svc := srvc.Entry{
		Name:            fmt.Sprintf("k8s-%s%d", protocol, port),
		Description:     "Auto generate service for Kubernetes",
		Protocol:        protocol,
		SourcePort:      "",
		DestinationPort: fmt.Sprintf("%d", port),
	}

	if err := op.objs.Services.Edit("", svc); err != nil {
		return err
	}
	return nil
}

func (op *ServiceOp) Delete(name string) error {
	if err := op.objs.Services.Delete("", name); err != nil {
		return err
	}
	return nil
}
