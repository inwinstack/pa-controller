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
	"github.com/PaloAltoNetworks/pango/poli/security"
)

type Security interface {
	List() ([]string, error)
	Get(string) (*security.Entry, error)
	Set(string, string, []string) error
	Delete(string) error
}

type SecurityOp struct {
	policies *poli.FwPoli
}

var _ Security = &SecurityOp{}

func (op *SecurityOp) List() ([]string, error) {
	policies, err := op.policies.Security.GetList("")
	if err != nil {
		return nil, err
	}
	return policies, nil
}

func (op *SecurityOp) Get(name string) (*security.Entry, error) {
	entry, err := op.policies.Security.Get("", name)
	if err != nil {
		return nil, err
	}
	return &entry, nil
}

func (op *SecurityOp) Set(name, srcAddr string, services []string) error {
	entry := security.Entry{
		Name:                            name,
		Description:                     "Auto sync Security for Kubernetes.",
		SourceZones:                     []string{"untrust"},
		SourceAddresses:                 []string{"any"},
		SourceUsers:                     []string{"any"},
		HipProfiles:                     []string{"any"},
		DestinationZones:                []string{"AI public service network"},
		DestinationAddresses:            []string{srcAddr},
		Applications:                    []string{"any"},
		Services:                        services,
		Categories:                      []string{"any"},
		Action:                          "allow",
		IcmpUnreachable:                 false,
		DisableServerResponseInspection: false,
	}

	entry.Defaults()
	if err := op.policies.Security.Set("", entry); err != nil {
		return err
	}
	return nil
}

func (op *SecurityOp) Delete(name string) error {
	if err := op.policies.Security.Delete("", name); err != nil {
		return err
	}
	return nil
}
