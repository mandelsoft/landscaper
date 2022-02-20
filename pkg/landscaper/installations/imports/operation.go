// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package imports

import (
	"context"
	"fmt"
	"strings"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	lsv1alpha1helper "github.com/gardener/landscaper/apis/core/v1alpha1/helper"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	lstmpl "github.com/gardener/landscaper/pkg/landscaper/installations/template"
)

const (
	// TemplatingFailedReason is the reason that is defined during templating.
	TemplatingFailedReason = "ImportValidationFailed"
)

// ImportOperation templates the executions and handles the interaction with
// the import object.
type ImportOperation struct {
	*installations.Operation
}

// New creates a new executions operations object
func New(op *installations.Operation) *ImportOperation {
	return &ImportOperation{
		Operation: op,
	}
}

func (o *ImportOperation) Ensure(ctx context.Context) error {
	inst := o.Inst
	cond := lsv1alpha1helper.GetOrInitCondition(inst.Info.Status.Conditions, lsv1alpha1.ValidateImportsCondition)

	templateStateHandler := lstmpl.KubernetesStateHandler{
		KubeClient: o.Client(),
		Inst:       inst.Info,
	}
	//tmpl := template.New(gotemplate.New(o.BlobResolver, templateStateHandler), spiff.New(templateStateHandler))
	tmpl := lstmpl.New(templateStateHandler, o.BlobResolver)
	errors, bindings, err := tmpl.TemplateImportExecutions(
		lstmpl.NewBlueprintExecutionOptions(
			o.Context().External.InjectComponentDescriptorRef(inst.Info),
			inst.Blueprint,
			o.ComponentDescriptor,
			o.ResolvedComponentDescriptorList,
			inst.GetImports()))

	if err != nil {
		inst.MergeConditions(lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionFalse,
			TemplatingFailedReason, "Unable to template executions"))
		return fmt.Errorf("unable to template executions: %w", err)
	}

	for k, v := range bindings {
		inst.Imports[k] = v
	}
	if len(errors) == 0 {
		return nil
	}

	msg := strings.Join(errors, ", ")
	inst.MergeConditions(lsv1alpha1helper.UpdatedCondition(cond, lsv1alpha1.ConditionFalse,
		TemplatingFailedReason, msg))
	return fmt.Errorf("import validation failed: %w", fmt.Errorf("%s", msg))
}
