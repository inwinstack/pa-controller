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

package service

import (
	blendedv1 "github.com/inwinstack/blended/apis/inwinstack/v1"
	"github.com/inwinstack/pango/objs/srvc"
)

func (c *Controller) newServiceObject(svc *blendedv1.Service) *srvc.Entry {
	return &srvc.Entry{
		Name:            svc.Name,
		Protocol:        svc.Spec.Protocol,
		SourcePort:      svc.Spec.SourcePort,
		DestinationPort: svc.Spec.DestinationPort,
		Description:     svc.Spec.Description,
		Tags:            svc.Spec.Tags,
	}
}

func (c *Controller) updateServiceObject(svc *blendedv1.Service) error {
	entry := c.newServiceObject(svc)
	if err := c.srvc.Edit(c.cfg.Vsys, *entry); err != nil {
		return err
	}
	c.commit <- true
	return nil
}

func (c *Controller) deleteServiceObject(svc *blendedv1.Service) error {
	enrty, err := c.srvc.Get(c.cfg.Vsys, svc.Name)
	if len(enrty.Name) == 0 && err != nil {
		return nil
	}

	if err := c.srvc.Delete(c.cfg.Vsys, svc.Name); err != nil {
		return err
	}
	c.commit <- true
	return nil
}
