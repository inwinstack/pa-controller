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

package util

import (
	"time"

	"github.com/golang/glog"
)

type RetriableError struct {
	Err error
}

func (r RetriableError) Error() string { return r.Err.Error() }

func Retry(callback func() error, d time.Duration, attempts int) (err error) {
	for i := 0; i < attempts; i++ {
		err = callback()
		if err == nil {
			return nil
		}
		glog.Errorf("Error: %s, Retrying in %s. %d Retries remaining.", err, d, attempts-i)
		time.Sleep(d)
	}
	return err
}

func SubtractNowTime(t time.Time) time.Duration {
	return time.Now().Sub(t)
}
