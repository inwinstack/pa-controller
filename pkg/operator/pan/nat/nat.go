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

package nat

import (
	blendedv1 "github.com/inwinstack/blended/apis/inwinstack/v1"
	"github.com/inwinstack/pango/poli/nat"
)

func (c *Controller) newNatPolicy(n *blendedv1.NAT) *nat.Entry {
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

func (c *Controller) updateNatPolicy(nat *blendedv1.NAT) error {
	entry := c.newNatPolicy(nat)
	if err := c.fwNat.Edit(c.cfg.Vsys, *entry); err != nil {
		return err
	}
	c.commit <- true
	return nil
}

func (c *Controller) deleteNatPolicy(nat *blendedv1.NAT) error {
	enrty, err := c.fwNat.Get(c.cfg.Vsys, nat.Name)
	if len(enrty.Name) == 0 && err != nil {
		return nil
	}

	if err := c.fwNat.Delete(c.cfg.Vsys, nat.Name); err != nil {
		return err
	}
	c.commit <- true
	return nil
}
