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

package operator

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	blendedv1 "github.com/inwinstack/blended/apis/inwinstack/v1"
	blendedfake "github.com/inwinstack/blended/client/clientset/versioned/fake"
	"github.com/inwinstack/pa-controller/pkg/config"
	"github.com/inwinstack/pango"
	"github.com/inwinstack/pango/objs"
	"github.com/inwinstack/pango/objs/srvc"
	"github.com/inwinstack/pango/poli"
	"github.com/inwinstack/pango/poli/nat"
	"github.com/inwinstack/pango/poli/security"
	"github.com/stretchr/testify/assert"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	extensionsfake "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type customResource struct {
	Name       string
	Kind       string
	Group      string
	Plural     string
	Version    string
	Scope      apiextensionsv1beta1.ResourceScope
	ShortNames []string
}

func createCRD(clientset apiextensionsclientset.Interface, resource customResource) error {
	crdName := fmt.Sprintf("%s.%s", resource.Plural, resource.Group)
	crd := &apiextensionsv1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: crdName,
		},
		Spec: apiextensionsv1beta1.CustomResourceDefinitionSpec{
			Group:   resource.Group,
			Version: resource.Version,
			Scope:   resource.Scope,
			Names: apiextensionsv1beta1.CustomResourceDefinitionNames{
				Singular:   resource.Name,
				Plural:     resource.Plural,
				Kind:       resource.Kind,
				ShortNames: resource.ShortNames,
			},
		},
	}
	_, err := clientset.ApiextensionsV1beta1().CustomResourceDefinitions().Create(crd)
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			return fmt.Errorf("failed to create %s CRD. %+v", resource.Name, err)
		}
	}
	return nil
}

func TestOperator(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	fw := &pango.Firewall{
		Policies: &poli.FwPoli{
			Nat:      &nat.FwNat{},
			Security: &security.FwSecurity{},
		},
		Objects: &objs.FwObjs{
			Services: &srvc.FwSrvc{},
		},
	}
	cfg := &config.Config{Threads: 2, Retry: 5}
	blendedset := blendedfake.NewSimpleClientset()
	extensionsClient := extensionsfake.NewSimpleClientset()

	resources := []customResource{
		{
			Name:    "service",
			Plural:  "services",
			Kind:    reflect.TypeOf(blendedv1.Service{}).Name(),
			Group:   blendedv1.CustomResourceGroup,
			Version: blendedv1.Version,
			Scope:   apiextensionsv1beta1.ClusterScoped,
		},
		{
			Name:    "security",
			Plural:  "securities",
			Kind:    reflect.TypeOf(blendedv1.Security{}).Name(),
			Group:   blendedv1.CustomResourceGroup,
			Version: blendedv1.Version,
			Scope:   apiextensionsv1beta1.NamespaceScoped,
		},
		{
			Name:    "nat",
			Plural:  "nats",
			Kind:    reflect.TypeOf(blendedv1.NAT{}).Name(),
			Group:   blendedv1.CustomResourceGroup,
			Version: blendedv1.Version,
			Scope:   apiextensionsv1beta1.NamespaceScoped,
		},
	}
	for _, res := range resources {
		assert.Nil(t, createCRD(extensionsClient, res))
	}

	crds, err := extensionsClient.ApiextensionsV1beta1().CustomResourceDefinitions().List(metav1.ListOptions{})
	assert.Nil(t, err)
	assert.Equal(t, len(resources), len(crds.Items))

	op := New(cfg, fw, blendedset)
	assert.NotNil(t, op)
	assert.Nil(t, op.Run(ctx))

	cancel()
	op.Stop()
}
