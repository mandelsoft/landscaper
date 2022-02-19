// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package template

import (
	"encoding/json"
	"fmt"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/codec"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/gardener/landscaper/apis/core"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/pkg/api"
)

// SubinstallationExecutorOutput describes the output of deploy executor.
type SubinstallationExecutorOutput struct {
	Subinstallations []*lsv1alpha1.InstallationTemplate `json:"subinstallations"`
}

func (o SubinstallationExecutorOutput) MarshalJSON() ([]byte, error) {
	return json.Marshal(o)
}

func (o *SubinstallationExecutorOutput) UnmarshalJSON(data []byte) error {
	type helperStruct struct {
		Subinstallations []json.RawMessage `json:"subinstallations"`
	}
	rawList := &helperStruct{}
	if err := json.Unmarshal(data, rawList); err != nil {
		return err
	}

	out := SubinstallationExecutorOutput{
		Subinstallations: make([]*lsv1alpha1.InstallationTemplate, len(rawList.Subinstallations)),
	}
	for i, raw := range rawList.Subinstallations {
		instTmpl := lsv1alpha1.InstallationTemplate{}
		if _, _, err := api.Decoder.Decode(raw, nil, &instTmpl); err != nil {
			return fmt.Errorf("unable to decode installation template %d: %w", i, err)
		}
		out.Subinstallations[i] = &instTmpl
	}

	*o = out
	return nil
}

// ImportExecutorOutput describes the output of import executor.
type ImportExecutorOutput struct {
	Bindings map[string]interface{} `json:"bindings"`
	Errors   []string               `json:"errors"`
}

type TargetReference struct {
	// +optional
	Name string `json:"name,omitempty"`

	// +optional
	Import string `json:"import,omitempty"`

	// +optional
	Index *int `json:"index,omitempty"`
}

// DeployItemSpecification defines a execution element that is translated into a deployitem template for the execution object.
type DeployItemSpecification struct {
	// Name is the unique name of the execution.
	Name string `json:"name"`

	// DataType is the DeployItem type of the execution.
	Type core.DeployItemType `json:"type"`

	// Target is the target reference to the target import of the target the deploy item should deploy to.
	// +optional
	Target *TargetReference `json:"target,omitempty"`

	// Labels is the map of labels to be added to the deploy item.
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// ProviderConfiguration contains the type specific configuration for the execution.
	Configuration *runtime.RawExtension `json:"config"`

	// DependsOn lists deploy items that need to be executed before this one
	DependsOn []string `json:"dependsOn,omitempty"`
}

// DeployExecutorOutput describes the output of deploy executor.
type DeployExecutorOutput struct {
	DeployItems []DeployItemSpecification `json:"deployItems"`
}

// ExportExecutorOutput describes the output of export executor.
type ExportExecutorOutput struct {
	Exports map[string]interface{} `json:"exports"`
}

func serializeComponentDescriptor(cd *cdv2.ComponentDescriptor) (interface{}, error) {
	if cd == nil {
		return nil, nil
	}
	data, err := codec.Encode(cd)
	if err != nil {
		return nil, err
	}
	var val interface{}
	if err := json.Unmarshal(data, &val); err != nil {
		return nil, err
	}
	return val, nil
}

func serializeComponentDescriptorList(cd *cdv2.ComponentDescriptorList) (interface{}, error) {
	if cd == nil {
		return nil, nil
	}
	data, err := codec.Encode(cd)
	if err != nil {
		return nil, err
	}
	var val interface{}
	if err := json.Unmarshal(data, &val); err != nil {
		return nil, err
	}
	return val, nil
}
