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

package k8sutil

import (
	"reflect"

	"github.com/inwinstack/pa-controller/pkg/constants"
	"github.com/thoas/go-funk"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func MakeNeedToUpdate(meta *metav1.ObjectMeta, old, new interface{}) {
	if !reflect.DeepEqual(old, new) {
		if meta.Annotations == nil {
			meta.Annotations = map[string]string{}
		}
		if _, ok := meta.Annotations[constants.NeedUpdateKey]; !ok {
			meta.Annotations[constants.NeedUpdateKey] = "true"
		}
	}
}

func IsNeedToUpdate(meta metav1.ObjectMeta) bool {
	_, ok := meta.Annotations[constants.NeedUpdateKey]
	return ok
}

func AddFinalizer(meta *metav1.ObjectMeta, finalizer string) {
	if !funk.ContainsString(meta.Finalizers, finalizer) {
		meta.Finalizers = append(meta.Finalizers, finalizer)
	}
}

func RemoveFinalizer(meta *metav1.ObjectMeta, finalizer string) {
	meta.Finalizers = funk.FilterString(meta.Finalizers, func(s string) bool {
		return s != finalizer
	})
}
