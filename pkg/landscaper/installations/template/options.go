// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package template

import (
	"fmt"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/landscaper/pkg/landscaper/templating"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
	"github.com/gardener/landscaper/pkg/utils"
)

// ExecutionOptions describes the base options for templating of all executions.
type ExecutionOptions struct {
	ValuesFunc           func() (map[string]interface{}, error)
	Installation         *lsv1alpha1.Installation
	Blueprint            *blueprints.Blueprint
	ComponentDescriptor  *cdv2.ComponentDescriptor
	ComponentDescriptors *cdv2.ComponentDescriptorList
}

// NewExecutionOptions create new basic blueprint execution options
func NewExecutionOptions(installation *lsv1alpha1.Installation, blueprint *blueprints.Blueprint, cd *cdv2.ComponentDescriptor, cdList *cdv2.ComponentDescriptorList, values ...func() (map[string]interface{}, error)) ExecutionOptions {
	o := ExecutionOptions{
		Installation:         installation,
		Blueprint:            blueprint,
		ComponentDescriptor:  cd,
		ComponentDescriptors: cdList,
	}
	if len(values) > 0 && values[0] != nil {
		o.ValuesFunc = values[0]
	} else {
		o.ValuesFunc = o.Values
	}
	return o
}

func (o *ExecutionOptions) TemplateContext(values map[string]interface{}) (*templating.TemplateContext, error) {
	effValues, err := o.ValuesFunc()
	if err != nil {
		return nil, err
	}
	if values != nil {
		for k, v := range values {
			effValues[k] = v
		}
	}
	return &templating.TemplateContext{
		Blueprint: o.Blueprint,
		Cd:        o.ComponentDescriptor,
		CdList:    o.ComponentDescriptors,
		Values:    effValues,
	}, nil
}

func (o *ExecutionOptions) Values() (map[string]interface{}, error) {
	// marshal and unmarshal resolved component descriptor
	component, err := serializeComponentDescriptor(o.ComponentDescriptor)
	if err != nil {
		return nil, fmt.Errorf("error during serializing of the resolved components: %w", err)
	}
	components, err := serializeComponentDescriptorList(o.ComponentDescriptors)
	if err != nil {
		return nil, fmt.Errorf("error during serializing of the component descriptor: %w", err)
	}

	values := map[string]interface{}{
		"cd":         component,
		"components": components,
	}

	// add blueprint and component descriptor ref information to the input values
	if o.Installation != nil {
		blueprintDef, err := utils.JSONSerializeToGenericObject(o.Installation.Spec.Blueprint)
		if err != nil {
			return nil, fmt.Errorf("unable to serialize the blueprint definition")
		}
		values["blueprint"] = blueprintDef
		values["blueprintDef"] = blueprintDef

		if o.Installation.Spec.ComponentDescriptor != nil {
			cdDef, err := utils.JSONSerializeToGenericObject(o.Installation.Spec.ComponentDescriptor)
			if err != nil {
				return nil, fmt.Errorf("unable to serialize the component descriptor definition")
			}
			values["componentDescriptorDef"] = cdDef
		}
	}
	return values, nil
}

// BlueprintExecutionOptions describes the base options for templating of all blueprint executions.
type BlueprintExecutionOptions struct {
	ExecutionOptions
	Imports map[string]interface{}
}

// NewBlueprintExecutionOptions create new basic blueprint execution options
func NewBlueprintExecutionOptions(installation *lsv1alpha1.Installation, blueprint *blueprints.Blueprint, cd *cdv2.ComponentDescriptor, cdList *cdv2.ComponentDescriptorList, imports map[string]interface{}) BlueprintExecutionOptions {
	o := BlueprintExecutionOptions{
		ExecutionOptions: NewExecutionOptions(installation, blueprint, cd, cdList),
		Imports:          imports,
	}
	o.ValuesFunc = o.Values
	return o
}

func (o *BlueprintExecutionOptions) Values() (map[string]interface{}, error) {
	values, err := o.ExecutionOptions.Values()
	if err != nil {
		return nil, err
	}
	values["imports"] = o.Imports
	return values, nil
}

////////////////////////////////////////////////////////////////////////////////

// DeployExecutionOptions describes the options for templating the deploy executions.
type DeployExecutionOptions struct {
	BlueprintExecutionOptions
}

func NewDeployExecutionOptions(base BlueprintExecutionOptions) DeployExecutionOptions {
	o := DeployExecutionOptions{
		BlueprintExecutionOptions: base,
	}
	o.ValuesFunc = o.Values
	return o
}

func (o *DeployExecutionOptions) Values() (map[string]interface{}, error) {
	return o.BlueprintExecutionOptions.Values()
}

////////////////////////////////////////////////////////////////////////////////

// ExportExecutionOptions describes the options for templating the deploy executions.
type ExportExecutionOptions struct {
	BlueprintExecutionOptions
	Exports map[string]interface{}
}

func NewExportExecutionOptions(base BlueprintExecutionOptions, exports map[string]interface{}) ExportExecutionOptions {
	o := ExportExecutionOptions{
		BlueprintExecutionOptions: base,
		Exports:                   exports,
	}
	o.ValuesFunc = o.Values
	return o
}

func (o *ExportExecutionOptions) Values() (map[string]interface{}, error) {
	values, err := o.BlueprintExecutionOptions.Values()
	if err != nil {
		return nil, err
	}
	values["values"] = o.Exports

	for k, v := range o.Exports {
		values[k] = v
	}
	return values, nil
}
