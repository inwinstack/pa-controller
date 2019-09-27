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

package ha

import (
	"context"
	"time"

	"github.com/inwinstack/pango/util"
)

const defaultSyncSecond = time.Second * 30

type Callbacks struct {
	OnActive  func(*util.HighAvailability)
	OnPassive func()
	OnFail    func(err error)
}

type Inspector struct {
	fw        util.XapiClient
	callbacks *Callbacks
	duration  time.Duration
}

func NewInspector(fw util.XapiClient, duration int, callbacks *Callbacks) *Inspector {
	syncSecond := defaultSyncSecond
	if duration > 30 {
		syncSecond = time.Second * time.Duration(duration)
	}
	return &Inspector{
		fw:        fw,
		duration:  syncSecond,
		callbacks: callbacks,
	}
}

func (i *Inspector) Run(ctx context.Context) error {
	if err := i.getStatus(); err != nil {
		return err
	}
	go i.startTicker(ctx.Done())
	return nil
}

func (i *Inspector) getStatus() error {
	status, err := i.fw.GetHighAvailabilityStatus()
	if err != nil {
		i.callbacks.OnFail(err)
		return err
	}

	if status.Enable == "yes" {
		if status.Group.Local.State == "active" {
			i.callbacks.OnActive(status)
			return nil
		}
		i.callbacks.OnPassive()
	}
	return nil
}

func (i *Inspector) startTicker(stopCh <-chan struct{}) {
	ticker := time.NewTicker(i.duration)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := i.getStatus(); err != nil {
				i.callbacks.OnFail(err)
			}
		case <-stopCh:
			return
		}
	}
}
