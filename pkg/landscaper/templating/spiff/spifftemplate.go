// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package spiff

import (
	"context"
	"fmt"

	"github.com/mandelsoft/spiff/spiffing"
	spiffyaml "github.com/mandelsoft/spiff/yaml"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"sigs.k8s.io/yaml"

	"github.com/gardener/landscaper/pkg/landscaper/templating"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

// Templater describes the spiff template implementation for execution templater.
type Templater struct {
}

var _ templating.Templater = (*Templater)(nil)

// New creates a new spiff execution templater.
func New() *Templater {
	return &Templater{}
}

func (t Templater) Type() lsv1alpha1.TemplateType {
	return lsv1alpha1.SpiffTemplateType
}

// Execute is the GoTemplate executor, if statedata == nil, no state handling is done
func (t *Templater) Execute(name string, templatedata, statedata []byte, tctx *templating.TemplateContext, result interface{}) (stateresult []byte, err error) {
	rawTemplate, err := spiffyaml.Unmarshal("template "+name, templatedata)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	defer ctx.Done()

	var stateNode []spiffyaml.Node

	if statedata != nil {
		if len(statedata) == 0 {
			stateNode = []spiffing.Node{spiffyaml.NewNode(map[string]interface{}{}, "state")}
		} else {
			node, err := spiffyaml.Unmarshal("state "+name, statedata)
			if err != nil {
				return nil, fmt.Errorf("unable to load state: %w", err)
			}
			stateNode = []spiffing.Node{node}
		}
	}

	functions := spiffing.NewFunctions()
	LandscaperSpiffFuncs(functions, tctx)

	binding := map[string]interface{}{
		"values": tctx.Values,
	}
	for k, v := range tctx.Values {
		binding[k] = v
	}

	var fs vfs.FileSystem
	var mode = 0
	if tctx.Blueprint != nil {
		fs = tctx.Blueprint.Fs
		if fs != nil {
			mode = spiffing.MODE_FILE_ACCESS
		}
	}

	spiff, err := spiffing.New().WithMode(mode).WithFunctions(functions).WithFileSystem(fs).WithValues(binding)
	if err != nil {
		return nil, fmt.Errorf("unable to init spiff templater: %w", err)
	}

	res, err := spiff.Cascade(rawTemplate, nil, stateNode...)
	if err != nil {
		return nil, err
	}

	stateresult, err = spiffyaml.Marshal(spiff.DetermineState(res))
	if err != nil {
		return nil, fmt.Errorf("unable to marshal state: %w", err)
	}
	if len(stateresult) == 0 {
		stateresult = nil
	}

	data, err := spiffyaml.Marshal(res)
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(data, result); err != nil {
		return stateresult, err
	}
	return stateresult, nil
}
