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

// Templater implements all available template executors.
type Templater struct {
	impl map[lsv1alpha1.TemplateType]ExecutionTemplater
}

// New creates a new instance of a templater.
func New(templaters ...ExecutionTemplater) *Templater {
	t := &Templater{
		impl: make(map[lsv1alpha1.TemplateType]ExecutionTemplater),
	}
	for _, templater := range templaters {
		t.impl[templater.Type()] = templater
	}
	return t
}

// ExecutionTemplater describes a implementation for a template execution
type ExecutionTemplater interface {
	// Type returns the type of the templater.
	Type() lsv1alpha1.TemplateType
	// TemplateImportExecutions templates an import executor and return a list of error messages.
	TemplateImportExecutions(tmplExec lsv1alpha1.TemplateExecutor,
		blueprint *blueprints.Blueprint,
		cd *cdv2.ComponentDescriptor,
		cdList *cdv2.ComponentDescriptorList,
		values map[string]interface{}) (*ImportExecutorOutput, error)
	// TemplateSubinstallationExecutions templates a subinstallation executor and return a list of installations templates.
	TemplateSubinstallationExecutions(tmplExec lsv1alpha1.TemplateExecutor,
		blueprint *blueprints.Blueprint,
		cd *cdv2.ComponentDescriptor,
		cdList *cdv2.ComponentDescriptorList,
		values map[string]interface{}) (*SubinstallationExecutorOutput, error)
	// TemplateDeployExecutions templates a deploy executor and return a list of deployitem templates.
	TemplateDeployExecutions(tmplExec lsv1alpha1.TemplateExecutor,
		blueprint *blueprints.Blueprint,
		cd *cdv2.ComponentDescriptor,
		cdList *cdv2.ComponentDescriptorList,
		values map[string]interface{}) (*DeployExecutorOutput, error)
	// TemplateExportExecutions templates a export executor.
	// It return the exported data as key value map where the key is the name of the export.
	TemplateExportExecutions(tmplExec lsv1alpha1.TemplateExecutor,
		blueprint *blueprints.Blueprint,
		descriptor *cdv2.ComponentDescriptor,
		cdList *cdv2.ComponentDescriptorList,
		values map[string]interface{}) (*ExportExecutorOutput, error)
}

func (o *Templater) TemplateImportExecutions(opts BlueprintExecutionOptions) ([]string, map[string]interface{}, error) {
	values, err := opts.Values()
	if err != nil {
		return nil, nil, err
	}

	errorList := []string{}
	bindings := map[string]interface{}{}

	for _, tmplExec := range opts.Blueprint.Info.ImportExecutions {
		impl, ok := o.impl[tmplExec.Type]
		if !ok {
			return nil, nil, fmt.Errorf("unknown template type %s", tmplExec.Type)
		}

		output, err := impl.TemplateImportExecutions(tmplExec, opts.Blueprint, opts.ComponentDescriptor, opts.ComponentDescriptors, values)
		if err != nil {
			return nil, nil, err
		}
		if output.Bindings != nil {
			var imports map[string]interface{}
			imp := values["imports"]
			if imports == nil {
				imports = map[string]interface{}{}
				values["imports"] = imports
			} else {
				imports = imp.(map[string]interface{})
			}
			for k, v := range output.Bindings {
				bindings[k] = v
				imports[k] = v
			}
		}
		if len(output.Errors) != 0 {
			errorList = append(errorList, output.Errors...)
			break
		}
	}

	return errorList, bindings, nil
}

// TemplateSubinstallationExecutions templates all subinstallation executions and
// returns a aggregated list of all templated installation templates.
func (o *Templater) TemplateSubinstallationExecutions(opts DeployExecutionOptions) ([]*lsv1alpha1.InstallationTemplate, error) {
	values, err := opts.Values()
	if err != nil {
		return nil, err
	}
	installationTemplates := make([]*lsv1alpha1.InstallationTemplate, 0)
	for _, tmplExec := range opts.Blueprint.Info.SubinstallationExecutions {
		impl, ok := o.impl[tmplExec.Type]
		if !ok {
			return nil, fmt.Errorf("unknown template type %s", tmplExec.Type)
		}

		output, err := impl.TemplateSubinstallationExecutions(tmplExec, opts.Blueprint, opts.ComponentDescriptor, opts.ComponentDescriptors, values)
		if err != nil {
			return nil, err
		}
		if output.Subinstallations == nil {
			continue
		}
		installationTemplates = append(installationTemplates, output.Subinstallations...)
	}

	return installationTemplates, nil
}

// TemplateDeployExecutions templates all deploy executions and returns a aggregated list of all templated deploy item templates.
func (o *Templater) TemplateDeployExecutions(opts DeployExecutionOptions) ([]DeployItemSpecification, error) {

	values, err := opts.Values()
	if err != nil {
		return nil, err
	}

	deployItemTemplateList := []DeployItemSpecification{}
	for _, tmplExec := range opts.Blueprint.Info.DeployExecutions {
		impl, ok := o.impl[tmplExec.Type]
		if !ok {
			return nil, fmt.Errorf("unknown template type %s", tmplExec.Type)
		}

		output, err := impl.TemplateDeployExecutions(tmplExec, opts.Blueprint, opts.ComponentDescriptor, opts.ComponentDescriptors, values)
		if err != nil {
			return nil, err
		}
		if output.DeployItems == nil {
			continue
		}
		deployItemTemplateList = append(deployItemTemplateList, output.DeployItems...)
	}

	return deployItemTemplateList, nil
}

// TemplateExportExecutions templates all exports.
func (o *Templater) TemplateExportExecutions(opts ExportExecutionOptions) (map[string]interface{}, error) {
	values, err := opts.Values()
	if err != nil {
		return nil, err
	}
	exportData := make(map[string]interface{})
	for _, tmplExec := range opts.Blueprint.Info.ExportExecutions {

		impl, ok := o.impl[tmplExec.Type]
		if !ok {
			return nil, fmt.Errorf("unknown template type %s", tmplExec.Type)
		}

		output, err := impl.TemplateExportExecutions(tmplExec, opts.Blueprint, opts.ComponentDescriptor, opts.ComponentDescriptors, values)
		if err != nil {
			return nil, err
		}
		exportData = utils.MergeMaps(exportData, output.Exports)
	}

	return exportData, nil
}