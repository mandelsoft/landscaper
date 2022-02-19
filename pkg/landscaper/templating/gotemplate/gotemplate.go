// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package gotemplate

import (
	"bytes"
	"encoding/json"
	"fmt"
	gotmpl "text/template"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/ctf"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/pkg/errors"
	"sigs.k8s.io/yaml"

	"github.com/gardener/landscaper/pkg/landscaper/templating"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
)

const (
	recursionMaxNums = 100
)

// Templater is the go template implementation for landscaper templating.
type Templater struct {
	blobResolver ctf.BlobResolver
}

var _ templating.Templater = (*Templater)(nil)

// New creates a new go template execution templater.
func New(blobResolver ctf.BlobResolver) *Templater {
	return &Templater{
		blobResolver: blobResolver,
	}
}

type TemplateExecution struct {
	funcMap       map[string]interface{}
	blueprint     *blueprints.Blueprint
	includedNames map[string]int
}

func NewTemplateExecution(blueprint *blueprints.Blueprint, cd *cdv2.ComponentDescriptor, cdList *cdv2.ComponentDescriptorList, blobResolver ctf.BlobResolver) *TemplateExecution {
	t := &TemplateExecution{
		funcMap:       LandscaperTplFuncMap(blueprint.Fs, cd, cdList, blobResolver),
		blueprint:     blueprint,
		includedNames: map[string]int{},
	}
	if blueprint != nil {
		t.funcMap["include"] = t.include
	}
	return t
}

func (te *TemplateExecution) include(name string, binding interface{}) (string, error) {
	if v, ok := te.includedNames[name]; ok {
		if v > recursionMaxNums {
			return "", errors.Wrapf(fmt.Errorf("unable to execute template"), "rendering template has a nested reference name: %s", name)
		}
		te.includedNames[name]++
	} else {
		te.includedNames[name] = 1
	}
	data, err := vfs.ReadFile(te.blueprint.Fs, name)
	if err != nil {
		return "", errors.Wrapf(err, "unable to read include file %q", name)
	}
	res, err := te.Execute(data, binding)
	te.includedNames[name]--
	return string(res), err
}

func (te *TemplateExecution) Execute(template []byte, binding interface{}) ([]byte, error) {
	var rawTemplate string
	if err := json.Unmarshal(template, &rawTemplate); err != nil {
		return nil, err
	}
	tmpl, err := gotmpl.New("execution").
		Funcs(LandscaperSprigFuncMap()).Funcs(te.funcMap).
		Option("missingkey=zero").
		Parse(rawTemplate)
	if err != nil {
		return nil, err
	}

	data := bytes.NewBuffer([]byte{})
	if err := tmpl.Execute(data, binding); err != nil {
		return nil, err
	}
	return data.Bytes(), nil
}

// StateTemplateResult describes the result of go templating.
type StateTemplateResult struct {
	State json.RawMessage `json:"state"`
}

func (t Templater) Type() lsv1alpha1.TemplateType {
	return lsv1alpha1.GOTemplateType
}

func (t *Templater) TemplateExecution(rawTemplate []byte, tctx *templating.TemplateContext, binding map[string]interface{}) ([]byte, error) {
	if len(rawTemplate) == 0 {
		return []byte{}, nil
	}
	te := NewTemplateExecution(tctx.Blueprint, tctx.Cd, tctx.CdList, t.blobResolver)
	return te.Execute(rawTemplate, binding)
}

// Execute is the GoTemplate executor, if statedata == nil, no state handling is done
func (t *Templater) Execute(name string, templatedata, statedata []byte, tctx *templating.TemplateContext, result interface{}) (stateresult []byte, err error) {
	var state interface{}

	binding := map[string]interface{}{}
	for k, v := range tctx.Values {
		binding[k] = v
	}

	if statedata != nil {
		if len(statedata) == 0 {
			state = map[string]interface{}{}
		} else {
			if err := yaml.Unmarshal(statedata, &state); err != nil {
				return nil, fmt.Errorf("unable to load state: %w", err)
			}
		}
		binding["state"] = state
	}

	data, err := t.TemplateExecution(templatedata, tctx, binding)
	if err != nil {
		return nil, fmt.Errorf("unable to execute template: %w", err)
	}

	if statedata != nil {
		res := &StateTemplateResult{}
		if err := yaml.Unmarshal(data, res); err != nil {
			return nil, err
		}
		statedata = res.State
	}
	if err := yaml.Unmarshal(data, result); err != nil {
		return nil, fmt.Errorf("error while decoding templated execution: %w", err)
	}
	return statedata, nil
}
