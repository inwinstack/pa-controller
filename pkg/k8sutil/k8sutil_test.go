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
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestMakeNeedToUpdate(t *testing.T) {
	tests := []struct {
		old      *corev1.Namespace
		new      *corev1.Namespace
		expected bool
	}{
		{
			old: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test1",
				},
				Spec: corev1.NamespaceSpec{},
			},
			new: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test1",
				},
				Spec: corev1.NamespaceSpec{
					Finalizers: []corev1.FinalizerName{"test1"},
				},
			},
			expected: true,
		},
		{
			old: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test1",
				},
				Spec: corev1.NamespaceSpec{},
			},
			new: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test1",
				},
				Spec: corev1.NamespaceSpec{},
			},
			expected: false,
		},
	}

	for _, test := range tests {
		MakeNeedToUpdate(&test.new.ObjectMeta, test.old, test.new)
		actual := IsNeedToUpdate(test.new.ObjectMeta)
		assert.Equal(t, test.expected, actual)
	}
}

func TestAddFinalizer(t *testing.T) {
	test := struct {
		meta     *metav1.ObjectMeta
		expected []string
	}{
		meta: &metav1.ObjectMeta{
			Name: "test1",
		},
		expected: []string{"kubernetes"},
	}

	AddFinalizer(test.meta, "kubernetes")
	assert.Equal(t, test.expected, test.meta.Finalizers)
}

func TestRemoveFinalizer(t *testing.T) {
	test := struct {
		meta     *metav1.ObjectMeta
		expected []string
	}{
		meta: &metav1.ObjectMeta{
			Name:       "test1",
			Finalizers: []string{"test", "kubernetes"},
		},
		expected: []string{"test"},
	}

	RemoveFinalizer(test.meta, "kubernetes")
	assert.Equal(t, test.expected, test.meta.Finalizers)
}
