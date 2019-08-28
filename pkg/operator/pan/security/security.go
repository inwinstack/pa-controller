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

package security

import (
	blendedv1 "github.com/inwinstack/blended/apis/inwinstack/v1"
	"github.com/inwinstack/pango/poli/security"
)

func (c *Controller) newSecurityPolicy(sec *blendedv1.Security) *security.Entry {
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

func (c *Controller) isExistingSecurityPolicy(sec *blendedv1.Security) bool {
	if entry, err := c.fwSec.Get(c.cfg.Vsys, sec.Name); err == nil {
		if len(entry.Name) != 0 {
			return true
		}
	}
	return false
}

func (c *Controller) updateSecurityPolicy(sec *blendedv1.Security) error {
	entry := c.newSecurityPolicy(sec)
	if err := c.fwSec.Edit(c.cfg.Vsys, *entry); err != nil {
		return err
	}

	if err := c.fwSec.MoveGroup(c.cfg.Vsys, c.cfg.MoveType, c.cfg.MoveRule, *entry); err != nil {
		return err
	}
	c.commit <- true
	return nil
}

func (c *Controller) deleteSecurityPolicy(sec *blendedv1.Security) error {
	if !c.isExistingSecurityPolicy(sec) {
		return nil
	}

	if err := c.fwSec.Delete(c.cfg.Vsys, sec.Name); err != nil {
		return err
	}
	c.commit <- true
	return nil
}
