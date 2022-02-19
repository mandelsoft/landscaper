// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package template

import (
	"fmt"

	"github.com/gardener/component-spec/bindings-go/ctf"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/landscaper/templating"
	"github.com/gardener/landscaper/pkg/landscaper/templating/gotemplate"
	"github.com/gardener/landscaper/pkg/landscaper/templating/spiff"
)

// BasicTemplater implements all available template executors.
type BasicTemplater struct {
	impl map[lsv1alpha1.TemplateType]templating.Templater
}

// NewBasic creates a new instance of a templater.
func NewBasic(blobResolver ctf.BlobResolver) *BasicTemplater {
	templaters := []templating.Templater{gotemplate.New(blobResolver), spiff.New()}
	t := &BasicTemplater{
		impl: make(map[lsv1alpha1.TemplateType]templating.Templater),
	}
	for _, templater := range templaters {
		t.impl[templater.Type()] = templater
	}
	return t
}

func (t *BasicTemplater) Execute(kind string, typ lsv1alpha1.TemplateType, name string, template []byte, state []byte, tctx *templating.TemplateContext, output interface{}) ([]byte, error) {
	impl, ok := t.impl[typ]
	if !ok {
		return nil, fmt.Errorf("%s execution %q: unknown template type %q", kind, name, typ)
	}
	return impl.Execute(name, template, state, tctx, output)
}
