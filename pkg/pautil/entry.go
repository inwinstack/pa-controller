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
	"github.com/PaloAltoNetworks/pango/objs/srvc"
	"github.com/PaloAltoNetworks/pango/poli/nat"
	"github.com/PaloAltoNetworks/pango/poli/security"
	inwinv1 "github.com/inwinstack/blended/apis/inwinstack/v1"
)

func ToSecurityEntry(sec *inwinv1.Security) *security.Entry {
	entry := &security.Entry{
		Name:                            sec.Name,
		Type:                            sec.Spec.Type,
		Description:                     sec.Spec.Description,
		Tags:                            sec.Spec.Tags,
		SourceZones:                     sec.Spec.SourceZones,
		SourceAddresses:                 sec.Spec.SourceAddresses,
		NegateSource:                    sec.Spec.NegateSource,
		SourceUsers:                     sec.Spec.SourceUsers,
		HipProfiles:                     sec.Spec.HipProfiles,
		DestinationZones:                sec.Spec.DestinationZones,
		DestinationAddresses:            sec.Spec.DestinationAddresses,
		NegateDestination:               sec.Spec.NegateDestination,
		Applications:                    sec.Spec.Applications,
		Services:                        sec.Spec.Services,
		Categories:                      sec.Spec.Categories,
		Action:                          sec.Spec.Action,
		LogSetting:                      sec.Spec.LogSetting,
		LogStart:                        sec.Spec.LogStart,
		LogEnd:                          sec.Spec.LogEnd,
		Disabled:                        sec.Spec.Disabled,
		Schedule:                        sec.Spec.Schedule,
		IcmpUnreachable:                 sec.Spec.IcmpUnreachable,
		DisableServerResponseInspection: sec.Spec.DisableServerResponseInspection,
		Group:                           sec.Spec.Group,
		Targets:                         sec.Spec.Targets,
		NegateTarget:                    sec.Spec.NegateTarget,
		Virus:                           sec.Spec.Virus,
		Spyware:                         sec.Spec.Spyware,
		Vulnerability:                   sec.Spec.Vulnerability,
		UrlFiltering:                    sec.Spec.URLFiltering,
		FileBlocking:                    sec.Spec.FileBlocking,
		WildFireAnalysis:                sec.Spec.WildFireAnalysis,
		DataFiltering:                   sec.Spec.DataFiltering,
	}
	entry.Defaults()
	return entry
}

func ToNatEntry(n *inwinv1.NAT) *nat.Entry {
	entry := &nat.Entry{
		Name:                           n.Name,
		Description:                    n.Spec.Description,
		Type:                           n.Spec.Type,
		SourceZones:                    n.Spec.SourceZones,
		DestinationZone:                n.Spec.DestinationZone,
		ToInterface:                    n.Spec.ToInterface,
		Service:                        n.Spec.Service,
		SourceAddresses:                n.Spec.SourceAddresses,
		DestinationAddresses:           n.Spec.DestinationAddresses,
		SatType:                        n.Spec.SatType,
		SatAddressType:                 n.Spec.SatAddressType,
		SatTranslatedAddresses:         n.Spec.SatTranslatedAddresses,
		SatInterface:                   n.Spec.SatInterface,
		SatIpAddress:                   n.Spec.SatIPAddress,
		SatFallbackType:                n.Spec.SatFallbackType,
		SatFallbackTranslatedAddresses: n.Spec.SatFallbackTranslatedAddresses,
		SatFallbackInterface:           n.Spec.SatFallbackInterface,
		SatFallbackIpType:              n.Spec.SatFallbackIPType,
		SatFallbackIpAddress:           n.Spec.SatFallbackIPAddress,
		SatStaticTranslatedAddress:     n.Spec.SatStaticTranslatedAddress,
		SatStaticBiDirectional:         n.Spec.SatStaticBiDirectional,
		DatType:                        n.Spec.DatType,
		DatAddress:                     n.Spec.DatAddress,
		DatPort:                        int(n.Spec.DatPort),
		DatDynamicDistribution:         n.Spec.DatDynamicDistribution,
		Disabled:                       n.Spec.Disabled,
		Targets:                        n.Spec.Targets,
		NegateTarget:                   n.Spec.NegateTarget,
		Tags:                           n.Spec.Tags,
	}
	entry.Defaults()
	return entry
}

func ToServiceEntry(svc *inwinv1.Service) *srvc.Entry {
	entry := &srvc.Entry{
		Name:            svc.Name,
		Protocol:        svc.Spec.Protocol,
		SourcePort:      svc.Spec.SourcePort,
		DestinationPort: svc.Spec.DestinationPort,
		Description:     svc.Spec.Description,
		Tags:            svc.Spec.Tags,
	}
	return entry
}
