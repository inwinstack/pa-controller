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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NATPolicyList is a list of NAT.
type NATPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []NATPolicy `json:"items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NATPolicy represents a Kubernetes NAT Custom Resource.
// The NATPolicy will be used as PAN-OS NAT policy.
type NATPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   NATPolicySpec `json:"spec"`
	Status Status        `json:"status,omitempty"`
}

// NATPolicySpec is the spec for a NAT resource.
type NATPolicySpec struct {
	Type                   string   `json:"type"`
	Description            string   `json:"description"`
	SourceZones            []string `json:"sourceZones"`
	DestinationZone        string   `json:"destinationZone"`
	ToInterface            string   `json:"toInterface"`
	Service                string   `json:"service"`
	SourceAddresses        []string `json:"sourceAddresses"`
	DestinationAddresses   []string `json:"destinationAddresses"`
	SatType                string   `json:"satType"`
	DatType                string   `json:"datType"`
	DatAddress             string   `json:"datAddress"`
	DatPort                int      `json:"datPort"`
	DatDynamicDistribution string   `json:"datDynamicDistribution"`
	Tags                   []string `json:"tags"`
}
