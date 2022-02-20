// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package template

import (
	"context"
	"fmt"

	"github.com/gardener/component-spec/bindings-go/ctf"
	"github.com/mandelsoft/vfs/pkg/vfs"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
	"github.com/gardener/landscaper/pkg/landscaper/templating"
	"github.com/gardener/landscaper/pkg/utils"
)

// Templater implements all available template executors.
type Templater struct {
	*BasicTemplater
	state GenericStateHandler
}

// New creates a new instance of a templater.
func New(state GenericStateHandler, resolver ctf.BlobResolver) *Templater {
	return &Templater{
		BasicTemplater: NewBasic(resolver),
		state:          state,
	}
}

func (t *Templater) getState(ctx context.Context, prefix string, name string) ([]byte, error) {
	if t.state == nil {
		return nil, nil // no state handling
	}
	data, err := t.state.Get(ctx, prefix+name)
	if err != nil {
		if err == StateNotFoundErr {
			return []byte{}, nil // initial state
		}
		return nil, err
	}

	return data, nil
}

func (t *Templater) storeState(ctx context.Context, prefix string, name string, data []byte) error {
	if t.state == nil {
		return nil
	}

	if len(data) == 0 {
		return nil
	}
	return t.state.Store(ctx, prefix+name, data)
}

func (t *Templater) execute(kind string, prefix string, tmplExec lsv1alpha1.TemplateExecutor, tctx *templating.TemplateContext, output interface{}) error {
	template, err := GetTemplateFromBlueprint(tmplExec, tctx.Blueprint)
	if err != nil {
		return fmt.Errorf("deployitem execution %q: %s", tmplExec.Name, err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var state []byte
	if t.state != nil && prefix != "" {
		state, err = t.getState(ctx, prefix, tmplExec.Name)
	}
	state, err = t.Execute(kind, tmplExec.Type, tmplExec.Name, template, state, tctx, output)
	if err != nil {
		return err
	}

	if t.state != nil && prefix != "" {
		err = t.storeState(ctx, prefix, tmplExec.Name, state)
		if err != nil {
			return fmt.Errorf("unable to store state for deployitem execution %q: %s", tmplExec.Name, err)
		}
	}
	return nil
}

func (o *Templater) TemplateImportExecutions(opts BlueprintExecutionOptions) ([]string, map[string]interface{}, error) {
	tctx, err := opts.TemplateContext(nil)
	if err != nil {
		return nil, nil, err
	}

	errorList := []string{}
	bindings := map[string]interface{}{}

	for _, tmplExec := range opts.Blueprint.Info.ImportExecutions {
		output := ImportExecutorOutput{}

		err := o.execute("import", "import", tmplExec, tctx, &output)
		if err != nil {
			return nil, nil, err
		}

		if output.Bindings != nil {
			var imports map[string]interface{}
			imp := tctx.Values["imports"]
			if imports == nil {
				imports = map[string]interface{}{}
				tctx.Values["imports"] = imports
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
	tctx, err := opts.TemplateContext(nil)
	if err != nil {
		return nil, err
	}

	installationTemplates := make([]*lsv1alpha1.InstallationTemplate, 0)
	for _, tmplExec := range opts.Blueprint.Info.SubinstallationExecutions {
		output := SubinstallationExecutorOutput{}
		err := o.execute("subinstallation", "deploy", tmplExec, tctx, &output)
		if err != nil {
			return nil, err
		}
		installationTemplates = append(installationTemplates, output.Subinstallations...)
	}

	return installationTemplates, nil
}

// TemplateDeployExecutions templates all deploy executions and returns a aggregated list of all templated deploy item templates.
func (o *Templater) TemplateDeployExecutions(opts DeployExecutionOptions) ([]DeployItemSpecification, error) {
	tctx, err := opts.TemplateContext(nil)
	if err != nil {
		return nil, err
	}

	deployItemTemplateList := []DeployItemSpecification{}
	for _, tmplExec := range opts.Blueprint.Info.DeployExecutions {
		output := DeployExecutorOutput{}
		err := o.execute("deployitem", "deploy", tmplExec, tctx, &output)
		if err != nil {
			return nil, err
		}
		deployItemTemplateList = append(deployItemTemplateList, output.DeployItems...)
	}

	return deployItemTemplateList, nil
}

// TemplateExportExecutions templates all exports.
func (o *Templater) TemplateExportExecutions(opts ExportExecutionOptions) (map[string]interface{}, error) {
	tctx, err := opts.TemplateContext(nil)
	if err != nil {
		return nil, err
	}

	exportData := make(map[string]interface{})
	for _, tmplExec := range opts.Blueprint.Info.ExportExecutions {
		output := ExportExecutorOutput{}
		err := o.execute("export", "export", tmplExec, tctx, &output)
		if err != nil {
			return nil, err
		}
		exportData = utils.MergeMaps(exportData, output.Exports)
	}

	return exportData, nil
}

func GetTemplateFromBlueprint(tmplExec lsv1alpha1.TemplateExecutor, blueprint *blueprints.Blueprint) ([]byte, error) {
	if len(tmplExec.Template.RawMessage) != 0 {
		return tmplExec.Template.RawMessage, nil
	}
	if len(tmplExec.File) != 0 {
		rawTemplateBytes, err := vfs.ReadFile(blueprint.Fs, tmplExec.File)
		if err != nil {
			return nil, err
		}
		return rawTemplateBytes, nil
	}
	return nil, fmt.Errorf("no template found")
}
