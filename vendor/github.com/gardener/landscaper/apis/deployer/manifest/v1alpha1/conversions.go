// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/runtime"

	lscore "github.com/gardener/landscaper/apis/core"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/gardener/landscaper/apis/deployer/manifest"
)

func addConversionFuncs(s *runtime.Scheme) error {
	if err := s.AddGeneratedConversionFunc((*ProviderConfiguration)(nil), (*manifest.ProviderConfiguration)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_ProviderConfiguration_To_manifest_ProviderConfiguration(a.(*ProviderConfiguration), b.(*manifest.ProviderConfiguration), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*manifest.ProviderConfiguration)(nil), (*ProviderConfiguration)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_manifest_ProviderConfiguration_To_v1alpha1_ProviderConfiguration(a.(*manifest.ProviderConfiguration), b.(*ProviderConfiguration), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*ProviderStatus)(nil), (*manifest.ProviderStatus)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_ProviderStatus_To_manifest_ProviderStatus(a.(*ProviderStatus), b.(*manifest.ProviderStatus), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*manifest.ProviderStatus)(nil), (*ProviderStatus)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_manifest_ProviderStatus_To_v1alpha1_ProviderStatus(a.(*manifest.ProviderStatus), b.(*ProviderStatus), scope)
	}); err != nil {
		return err
	}

	return nil
}

// Convert_v1alpha1_ProviderConfiguration_To_manifest_ProviderConfiguration is an autogenerated conversion function.
func Convert_v1alpha1_ProviderConfiguration_To_manifest_ProviderConfiguration(in *ProviderConfiguration, out *manifest.ProviderConfiguration, s conversion.Scope) error {
	out.Kubeconfig = in.Kubeconfig
	out.UpdateStrategy = manifest.UpdateStrategy(in.UpdateStrategy)
	if err := Convert_v1alpha1_HealthChecksConfiguration_To_manifest_HealthChecksConfiguration(&in.HealthChecks, &out.HealthChecks, s); err != nil {
		return err
	}
	if in.DeleteTimeout == nil {
		out.DeleteTimeout = nil
	} else {
		out.DeleteTimeout = &lscore.Duration{}
		if err := lsv1alpha1.Convert_v1alpha1_Duration_To_core_Duration(in.DeleteTimeout, out.DeleteTimeout, s); err != nil {
			return err
		}
	}
	if in.Manifests != nil {
		in, out := &in.Manifests, &out.Manifests
		*out = make([]manifest.Manifest, len(*in))
		for i := range *in {
			(*out)[i] = manifest.Manifest{
				Policy:   manifest.ManagePolicy,
				Manifest: (*in)[i],
			}
		}
	} else {
		out.Manifests = nil
	}
	return nil
}

// Convert_manifest_ProviderConfiguration_To_v1alpha1_ProviderConfiguration is an autogenerated conversion function.
func Convert_manifest_ProviderConfiguration_To_v1alpha1_ProviderConfiguration(in *manifest.ProviderConfiguration, out *ProviderConfiguration, s conversion.Scope) error {
	out.Kubeconfig = in.Kubeconfig
	out.UpdateStrategy = UpdateStrategy(in.UpdateStrategy)
	if err := Convert_manifest_HealthChecksConfiguration_To_v1alpha1_HealthChecksConfiguration(&in.HealthChecks, &out.HealthChecks, s); err != nil {
		return err
	}
	if in.DeleteTimeout == nil {
		out.DeleteTimeout = nil
	} else {
		out.DeleteTimeout = &lsv1alpha1.Duration{}
		if err := lsv1alpha1.Convert_core_Duration_To_v1alpha1_Duration(in.DeleteTimeout, out.DeleteTimeout, s); err != nil {
			return err
		}
	}
	if in.Manifests != nil {
		in, out := &in.Manifests, &out.Manifests
		*out = make([]*runtime.RawExtension, len(*in))
		for i := range *in {
			(*out)[i] = (*in)[i].Manifest
		}
	} else {
		out.Manifests = nil
	}
	return nil
}

// Convert_v1alpha1_ProviderStatus_To_manifest_ProviderStatus is an autogenerated conversion function.
func Convert_v1alpha1_ProviderStatus_To_manifest_ProviderStatus(in *ProviderStatus, out *manifest.ProviderStatus, s conversion.Scope) error {
	if in.ManagedResources != nil {
		in, out := &in.ManagedResources, &out.ManagedResources
		*out = make([]manifest.ManagedResourceStatus, len(*in))
		for i := range *in {
			tmpOut := &lscore.TypedObjectReference{}
			if err := lsv1alpha1.Convert_v1alpha1_TypedObjectReference_To_core_TypedObjectReference(&(*in)[i], tmpOut, s); err != nil {
				return err
			}
			(*out)[i] = manifest.ManagedResourceStatus{
				Policy:   manifest.ManagePolicy,
				Resource: *tmpOut,
			}
		}
	} else {
		out.ManagedResources = nil
	}
	return nil
}

// Convert_manifest_ProviderStatus_To_v1alpha1_ProviderStatus is an autogenerated conversion function.
func Convert_manifest_ProviderStatus_To_v1alpha1_ProviderStatus(in *manifest.ProviderStatus, out *ProviderStatus, s conversion.Scope) error {
	if in.ManagedResources != nil {
		in, out := &in.ManagedResources, &out.ManagedResources
		*out = make([]lsv1alpha1.TypedObjectReference, len(*in))
		for i := range *in {
			tmpOut := &lsv1alpha1.TypedObjectReference{}
			if err := lsv1alpha1.Convert_core_TypedObjectReference_To_v1alpha1_TypedObjectReference(&(*in)[i].Resource, tmpOut, s); err != nil {
				return err
			}
			(*out)[i] = *tmpOut
		}
	} else {
		out.ManagedResources = nil
	}
	return nil
}
