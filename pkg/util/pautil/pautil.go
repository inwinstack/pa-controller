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
	"github.com/PaloAltoNetworks/pango/poli/nat"
	"github.com/PaloAltoNetworks/pango/poli/security"
	"github.com/golang/glog"
)

func NewPAClient(host, user, password string) (*pango.Firewall, error) {
	client := &pango.Firewall{Client: pango.Client{
		Hostname: host,
		Username: user,
		Password: password,
		Logging:  pango.LogQuiet,
	}}

	if err := client.Initialize(); err != nil {
		return nil, err
	}
	glog.V(2).Infof("Get PA version: %s.", client.Versioning())
	return client, nil
}

func TestGetNATPolicys(client *pango.Firewall) ([]string, error) {
	policies, err := client.Policies.Nat.GetList("")
	if err != nil {
		return nil, err
	}
	for _, poli := range policies {
		p, _ := client.Policies.Nat.Get("", poli)
		glog.V(2).Infof("Show %s: %v.", poli, p)
	}
	return policies, nil
}

func TestGetSecurityPolicys(client *pango.Firewall) ([]string, error) {
	policies, err := client.Policies.Security.GetList("")
	if err != nil {
		return nil, err
	}
	for _, poli := range policies {
		p, _ := client.Policies.Security.Get("", poli)
		glog.V(2).Infof("Show %s: %s.", poli, p)
	}
	return policies, nil
}

func TestSetNATPolicy(client *pango.Firewall) error {
	entry := nat.Entry{
		Name:                 "Nat policy",
		Description:          "My NAT policy",
		Type:                 "ipv4",
		SourceZones:          []string{"untrust"},
		DestinationZone:      "trust",
		ToInterface:          "ethernet1/2",
		Service:              "any",
		SourceAddresses:      []string{"any"},
		DestinationAddresses: []string{"192.168.200.10"},
		SatType:              "none",
		DatType:              "destination-translation",
		DatAddress:           "140.9.1.100",
		DatPort:              1234,
		Tags:                 []string{"k8s-nat"},
	}

	err := client.Policies.Nat.Set("", entry)
	if err != nil {
		return err
	}
	return nil
}

func TestSetSecurityPolicy(client *pango.Firewall) error {
	entry := security.Entry{
		Name:                 "Security policy",
		Description:          "My security policy",
		Type:                 "universal",
		NegateSource:         false,
		NegateDestination:    false,
		Action:               "allow",
		SourceZones:          []string{"any"},
		SourceUsers:          []string{"any"},
		DestinationZones:     []string{"any"},
		DestinationAddresses: []string{"any"},
		Applications:         []string{"any"},
		Services:             []string{"any"},
		Categories:           []string{"any"},
		HipProfiles:          []string{"any"},
		Tags:                 []string{"k8s-nat"},
	}
	err := client.Policies.Security.Set("", entry)
	if err != nil {
		glog.V(2).Infof("Err: %s.", err)
		return err
	}
	return nil
}
