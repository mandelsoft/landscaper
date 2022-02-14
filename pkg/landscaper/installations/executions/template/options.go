// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package template

import (
	"fmt"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
	"github.com/gardener/landscaper/pkg/utils"
)

// BlueprintExecutionOptions describes the base options for templating of all blueprint executions.
type BlueprintExecutionOptions struct {
	Installation         *lsv1alpha1.Installation
	Blueprint            *blueprints.Blueprint
	ComponentDescriptor  *cdv2.ComponentDescriptor
	ComponentDescriptors *cdv2.ComponentDescriptorList
}

// NewBlueprintExecutionOptions create new basic blueprint execution options
func NewBlueprintExecutionOptions(installation *lsv1alpha1.Installation, blueprint *blueprints.Blueprint, cd *cdv2.ComponentDescriptor, cdList *cdv2.ComponentDescriptorList) BlueprintExecutionOptions {
	return BlueprintExecutionOptions{
		Installation:         installation,
		Blueprint:            blueprint,
		ComponentDescriptor:  cd,
		ComponentDescriptors: cdList,
	}
}

func (o *BlueprintExecutionOptions) Values() (map[string]interface{}, error) {
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

////////////////////////////////////////////////////////////////////////////////

// DeployExecutionOptions describes the options for templating the deploy executions.
type DeployExecutionOptions struct {
	BlueprintExecutionOptions
	Imports map[string]interface{}
}

func NewDeployExecutionOptions(base BlueprintExecutionOptions, imports map[string]interface{}) DeployExecutionOptions {
	return DeployExecutionOptions{
		BlueprintExecutionOptions: base,
		Imports:                   imports,
	}
}

func (o *DeployExecutionOptions) Values() (map[string]interface{}, error) {
	values, err := o.BlueprintExecutionOptions.Values()
	if err != nil {
		return nil, err
	}
	values["values"] = o.Imports
	values["imports"] = o.Imports
	return values, nil
}

////////////////////////////////////////////////////////////////////////////////

// ExportExecutionOptions describes the options for templating the deploy executions.
type ExportExecutionOptions struct {
	BlueprintExecutionOptions
	Exports map[string]interface{}
}

func NewExportExecutionOptions(base BlueprintExecutionOptions, exports map[string]interface{}) ExportExecutionOptions {
	return ExportExecutionOptions{
		BlueprintExecutionOptions: base,
		Exports:                   exports,
	}
}

func (o *ExportExecutionOptions) Values() (map[string]interface{}, error) {
	values, err := o.BlueprintExecutionOptions.Values()
	if err != nil {
		return nil, err
	}
	values["values"] = o.Exports
	values["exports"] = o.Exports

	for k, v := range o.Exports {
		values[k] = v
	}
	return values, nil
}
