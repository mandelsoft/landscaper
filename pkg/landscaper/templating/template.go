// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package templating

import (
	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
)

// TemplateContext holds the context information for template processing
type TemplateContext struct {
	Blueprint *blueprints.Blueprint
	Cd        *cdv2.ComponentDescriptor
	CdList    *cdv2.ComponentDescriptorList
	Values    map[string]interface{}
}

// Templater describes a implementation for a template execution
type Templater interface {
	// Type returns the type of the templater.
	Type() lsv1alpha1.TemplateType

	// Execute evaluates a template for a given state and context to a result structure and updated state.
	// statedata==nil: no state handling
	// statedata=[]:   initial state
	Execute(name string, templatedata, statedata []byte, tctx *TemplateContext, result interface{}) (stateresult []byte, err error)
}
