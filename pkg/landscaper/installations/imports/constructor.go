// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package imports

import (
	"context"
	"fmt"

	"github.com/mandelsoft/spiff/spiffing"
	spiffyaml "github.com/mandelsoft/spiff/yaml"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/yaml"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/landscaper/dataobjects"
	"github.com/gardener/landscaper/pkg/landscaper/dataobjects/jsonpath"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	"github.com/gardener/landscaper/pkg/landscaper/installations/template"
)

// NewConstructor creates a new Import Constructor.
func NewConstructor(op *installations.Operation) *Constructor {
	return &Constructor{
		Operation: op,
		siblings:  op.Context().Siblings,
	}
}

// Construct loads all imported data from the data sources (either installations or the landscape config)
// and creates the imported configuration.
// The imported data is added to installation resource.
func (c *Constructor) Construct(ctx context.Context) error {
	var (
		inst    = c.Inst
		fldPath = field.NewPath(inst.Info.Name)
	)

	// read imports and construct internal templating imports
	importedDataObjects, err := c.GetImportedDataObjects(ctx) // returns a map mapping logical names to data objects
	if err != nil {
		return err
	}
	importedTargets, err := c.GetImportedTargets(ctx) // returns a map mapping logical names to targets
	if err != nil {
		return err
	}
	importedTargetLists, err := c.GetImportedTargetLists(ctx) // returns a map mapping logical names to target lists
	if err != nil {
		return err
	}
	importedComponentDescriptors, err := c.GetImportedComponentDescriptors(ctx) // returns a map mapping logical names to component descriptors
	if err != nil {
		return err
	}
	importedComponentDescriptorLists, err := c.GetImportedComponentDescriptorLists(ctx) // returns a map mapping logical names to lists of component descriptors
	if err != nil {
		return err
	}

	instImports, templatedDataMappings, err := c.templateDataMappings(fldPath, importedDataObjects, importedTargets, importedTargetLists, importedComponentDescriptors, importedComponentDescriptorLists) // returns a map mapping logical names to data content
	if err != nil {
		return err
	}

	// add additional imports and targets
	imports, err := c.constructImports(inst.Blueprint.Info.Imports, importedDataObjects, importedTargets, importedTargetLists, importedComponentDescriptors, importedComponentDescriptorLists, templatedDataMappings, fldPath)
	if err != nil {
		return err
	}

	c.SetTargetImports(importedTargets)
	c.SetTargetListImports(importedTargetLists)

	inst.SetImports(imports)
	inst.SetInstImports(instImports)
	inst.SetImportMappings(templatedDataMappings)
	return nil
}

// constructImports is an auxiliary function that can be called in a recursive manner to traverse the tree of conditional imports
func (c *Constructor) constructImports(
	importList lsv1alpha1.ImportDefinitionList,
	importedDataObjects map[string]*dataobjects.DataObject,
	importedTargets map[string]*dataobjects.Target,
	importedTargetLists map[string]*dataobjects.TargetList,
	importedComponentDescriptors map[string]*dataobjects.ComponentDescriptor,
	importedComponentDescriptorLists map[string]*dataobjects.ComponentDescriptorList,
	templatedDataMappings map[string]interface{},
	fldPath *field.Path) (map[string]interface{}, error) {

	imports := map[string]interface{}{}
	for _, def := range importList {
		var err error
		defPath := fldPath.Child(def.Name)
		switch def.Type {
		case lsv1alpha1.ImportTypeData:
			if val, ok := templatedDataMappings[def.Name]; ok {
				imports[def.Name] = val
			} else if val, ok := importedDataObjects[def.Name]; ok {
				imports[def.Name] = val.Data
			}
			if _, ok := imports[def.Name]; !ok {
				if def.Required != nil && !*def.Required {
					continue // don't throw an error if the import is not required
				}
				return nil, installations.NewImportNotFoundErrorf(nil, "blueprint defines import %q of type %s, which is not satisfied", def.Name, lsv1alpha1.ImportTypeData)
			}
			validator, err := c.JSONSchemaValidator(def.Schema.RawMessage)
			if err != nil {
				return imports, installations.NewErrorf(installations.SchemaValidationFailed, err, "%s: validator creation failed", defPath.String())
			}
			if err := validator.ValidateGoStruct(imports[def.Name]); err != nil {
				return imports, installations.NewErrorf(installations.SchemaValidationFailed, err, "%s: imported datatype does not have the expected schema", defPath.String())
			}
			if len(def.ConditionalImports) > 0 {
				// recursively check conditional imports
				conditionalImports, err := c.constructImports(def.ConditionalImports, importedDataObjects, importedTargets, importedTargetLists, importedComponentDescriptors, importedComponentDescriptorLists, templatedDataMappings, defPath)
				if err != nil {
					return nil, err
				}
				for k, v := range conditionalImports {
					imports[k] = v
				}
			}
			continue
		case lsv1alpha1.ImportTypeTarget:
			if val, ok := importedTargets[def.Name]; ok {
				imports[def.Name], err = val.GetData()
				if err != nil {
					return nil, installations.NewErrorf(installations.SchemaValidationFailed, err, "%s: imported target cannot be parsed", defPath.String())
				}
			}
			data, ok := imports[def.Name]
			if !ok {
				if def.Required != nil && !*def.Required {
					continue // don't throw an error if the import is not required
				}
				return nil, installations.NewImportNotFoundErrorf(nil, "blueprint defines import %q of type %s, which is not satisfied", def.Name, lsv1alpha1.ImportTypeTarget)
			}

			var targetType string
			if err := jsonpath.GetValue(".spec.type", data, &targetType); err != nil {
				return nil, installations.NewErrorf(installations.SchemaValidationFailed, err, "%s: imported target does not match the expected target template schema", defPath.String())
			}
			if def.TargetType != targetType {
				return nil, installations.NewErrorf(installations.SchemaValidationFailed, nil, "%s: imported target type is %s but expected %s", defPath.String(), targetType, def.TargetType)
			}
			continue
		case lsv1alpha1.ImportTypeTargetList:
			if val, ok := importedTargetLists[def.Name]; ok {
				imports[def.Name], err = val.GetData()
				if err != nil {
					return nil, installations.NewErrorf(installations.SchemaValidationFailed, err, "%s: imported target cannot be parsed", defPath.String())
				}
			}
			data, ok := imports[def.Name]
			if !ok {
				if def.Required != nil && !*def.Required {
					continue // don't throw an error if the import is not required
				}
				return nil, installations.NewImportNotFoundErrorf(nil, "blueprint defines import %q of type %s, which is not satisfied", def.Name, lsv1alpha1.ImportTypeTargetList)
			}

			var targetType string
			listData, ok := data.([]interface{})
			if !ok {
				return nil, installations.NewErrorf(installations.SchemaValidationFailed, nil, "%s: targetlist import is not a list", defPath.String())
			}
			for i, elem := range listData {
				if err := jsonpath.GetValue(".spec.type", elem, &targetType); err != nil {
					return nil, installations.NewErrorf(installations.SchemaValidationFailed, err, "%s: element at position %d of the imported targetlist does not match the expected target template schema", defPath.String(), i)
				}
				if def.TargetType != targetType {
					return nil, installations.NewErrorf(installations.SchemaValidationFailed, nil, "%s: type of the element at position %d of the imported targetlist is %s but expected %s", defPath.String(), i, targetType, def.TargetType)
				}
			}
			continue
		case lsv1alpha1.ImportTypeComponentDescriptor:
			if val, ok := importedComponentDescriptors[def.Name]; ok {
				imports[def.Name], err = val.GetData()
				if err != nil {
					return nil, installations.NewErrorf(installations.SchemaValidationFailed, err, "%s: imported component descriptor cannot be parsed", defPath.String())
				}
			}
			_, ok := imports[def.Name]
			if !ok {
				if def.Required != nil && !*def.Required {
					continue // don't throw an error if the import is not required
				}
				return nil, installations.NewImportNotFoundErrorf(nil, "blueprint defines import %q of type %s, which is not satisfied", def.Name, lsv1alpha1.ImportTypeComponentDescriptor)
			}
			continue
		case lsv1alpha1.ImportTypeComponentDescriptorList:
			if val, ok := importedComponentDescriptorLists[def.Name]; ok {
				imports[def.Name], err = val.GetData()
				if err != nil {
					return nil, installations.NewErrorf(installations.SchemaValidationFailed, err, "%s: imported component descriptor list cannot be parsed", defPath.String())
				}
			}
			_, ok := imports[def.Name]
			if !ok {
				if def.Required != nil && !*def.Required {
					continue // don't throw an error if the import is not required
				}
				return nil, installations.NewImportNotFoundErrorf(nil, "blueprint defines import %q of type %s, which is not satisfied", def.Name, lsv1alpha1.ImportTypeComponentDescriptorList)
			}
			continue
		default:
			return nil, fmt.Errorf("%s: unknown import type '%s'", defPath.String(), string(def.Type))
		}
	}

	return imports, nil
}

type DataMappingOutput struct {
	Mapping map[string]interface{} `json:"mapping"`
}

func (c *Constructor) templateDataMappings(
	fldPath *field.Path,
	importedDataObjects map[string]*dataobjects.DataObject,
	importedTargets map[string]*dataobjects.Target,
	importedTargetLists map[string]*dataobjects.TargetList,
	importedComponentDescriptors map[string]*dataobjects.ComponentDescriptor,
	importedComponentDescriptorLists map[string]*dataobjects.ComponentDescriptorList) (map[string]interface{}, map[string]interface{}, error) {

	_ = importedComponentDescriptorLists
	templateValues := map[string]interface{}{}
	for name, do := range importedDataObjects {
		templateValues[name] = do.Data
	}
	for name, target := range importedTargets {
		var err error
		templateValues[name], err = target.GetData()
		if err != nil {
			return nil, nil, fmt.Errorf("unable to get target data for import %s", name)
		}
	}
	for name, targetlist := range importedTargetLists {
		var err error
		templateValues[name], err = targetlist.GetData()
		if err != nil {
			return nil, nil, fmt.Errorf("unable to get targetlist data for import %s", name)
		}
	}
	for name, cd := range importedComponentDescriptors {
		var err error
		templateValues[name], err = cd.GetData()
		if err != nil {
			return nil, nil, fmt.Errorf("unable to get target data for import %s", name)
		}
	}

	imports := map[string]interface{}{}
	for k, v := range templateValues {
		imports[k] = v
	}
	templater := template.NewBasic(c.Operation.BlobResolver)

	opts := template.NewExecutionOptions(c.Operation.Inst.Info, c.Operation.Inst.Blueprint, c.Operation.ComponentDescriptor, c.Operation.ResolvedComponentDescriptorList)

	tctx, err := opts.TemplateContext(map[string]interface{}{
		"imports": templateValues,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("unable to construct template context: %w", err)
	}

	values := make(map[string]interface{})
	for _, tmplExec := range c.Inst.Info.Spec.ImportDataExecutions {
		output := DataMappingOutput{}
		_, err := templater.Execute("import mapping", tmplExec.Type, tmplExec.Name, tmplExec.Template.RawMessage, nil, tctx, &output)
		if err != nil {
			return nil, nil, err
		}

		if output.Mapping != nil {
			for k, v := range output.Mapping {
				templateValues[k] = v
				values[k] = v
			}
		}
	}

	spiff, err := spiffing.New().WithFunctions(spiffing.NewFunctions()).WithValues(templateValues)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to init spiff templater: %w", err)
	}
	for key, dataMapping := range c.Inst.Info.Spec.ImportDataMappings {
		impPath := fldPath.Child(key)

		tmpl, err := spiffyaml.Unmarshal(key, dataMapping.RawMessage)
		if err != nil {
			return nil, nil, fmt.Errorf("unable to parse import mapping template %s: %w", impPath.String(), err)
		}

		res, err := spiff.Cascade(tmpl, nil)
		if err != nil {
			return nil, nil, fmt.Errorf("unable to template import mapping template %s: %w", impPath.String(), err)
		}

		dataBytes, err := spiffyaml.Marshal(res)
		if err != nil {
			return nil, nil, fmt.Errorf("unable to marshal templated import mapping %s: %w", impPath.String(), err)
		}
		var data interface{}
		if err := yaml.Unmarshal(dataBytes, &data); err != nil {
			return nil, nil, fmt.Errorf("unable to unmarshal templated import mapping %s: %w", impPath.String(), err)
		}
		values[key] = data
	}
	return imports, values, nil
}
