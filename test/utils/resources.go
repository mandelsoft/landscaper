// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"path/filepath"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/golang/mock/gomock"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/projectionfs"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	componentsregistry "github.com/gardener/landscaper/pkg/landscaper/registry/components"
	"github.com/gardener/landscaper/test/utils/envtest"

	"github.com/gardener/landscaper/pkg/api"
	lsutils "github.com/gardener/landscaper/pkg/utils/landscaper"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	containerv1alpha1 "github.com/gardener/landscaper/apis/deployer/container/v1alpha1"
	kutil "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	k8smock "github.com/gardener/landscaper/controller-utils/pkg/kubernetes/mock"
	"github.com/gardener/landscaper/pkg/deployer/container"
	"github.com/gardener/landscaper/pkg/landscaper/blueprints"
	"github.com/gardener/landscaper/pkg/landscaper/installations"
	lsoperation "github.com/gardener/landscaper/pkg/landscaper/operation"
)

// TestInstallationConfig defines a installation configuration which can be used to create
// a test environment with a installation, a blueprint and a operation.
type TestInstallationConfig struct {
	// +optional
	MockClient *k8smock.MockClient
	// Defines the installation that should be used to create a blueprint and operations
	// If it is not defined a default one is created with the given name and namespace
	// +optional
	Installation *lsv1alpha1.Installation

	// Configures the default created installation
	InstallationName             string
	InstallationNamespace        string
	RemoteBlueprintComponentName string
	RemoteBlueprintResourceName  string
	RemoteBlueprintVersion       string
	RemoteBlueprintBaseURL       string

	BlueprintContentPath string
	// BlueprintFilePath defines the filepath to the blueprint definition.
	// Will be defaulted to <BlueprintContentPath>/blueprint.yaml if not defined.
	BlueprintFilePath string
}

// CreateTestInstallationResources creates a test environment with a installation, a blueprint and a operation.
// Should only be used for root installation as other installations may be created during runtime.
func CreateTestInstallationResources(op *lsoperation.Operation, cfg TestInstallationConfig) (*lsv1alpha1.Installation, *installations.Installation, *blueprints.Blueprint, *installations.Operation) {
	// apply defaults
	if len(cfg.BlueprintFilePath) == 0 {
		cfg.BlueprintFilePath = filepath.Join(cfg.BlueprintContentPath, "blueprint.yaml")
	}

	if cfg.MockClient != nil {
		cfg.MockClient.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).DoAndReturn(
			func(ctx context.Context, instList *lsv1alpha1.InstallationList, _ ...interface{}) error {
				*instList = lsv1alpha1.InstallationList{}
				return nil
			})
	}

	rootInst := cfg.Installation
	if rootInst == nil {
		rootInst = &lsv1alpha1.Installation{}
		rootInst.Name = cfg.InstallationName
		rootInst.Namespace = cfg.InstallationNamespace
		rootInst.Spec.ComponentDescriptor = LocalRemoteComponentDescriptorRef(cfg.RemoteBlueprintComponentName, cfg.RemoteBlueprintVersion, cfg.RemoteBlueprintBaseURL)
		rootInst.Spec.Blueprint = LocalRemoteBlueprintRef(cfg.RemoteBlueprintResourceName)
	}

	rootBlueprint := CreateBlueprintFromFile(cfg.BlueprintFilePath, cfg.BlueprintContentPath)

	rootIntInst, err := installations.New(rootInst, rootBlueprint)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	rootInstOp, err := installations.NewOperationBuilder(rootIntInst).WithOperation(op).Build(context.TODO())
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	return rootInst, rootIntInst, rootBlueprint, rootInstOp
}

// LocalRemoteComponentDescriptorRef creates a new default local remote component descriptor reference
func LocalRemoteComponentDescriptorRef(componentName, version, baseURL string) *lsv1alpha1.ComponentDescriptorDefinition {
	repoCtx, _ := cdv2.NewUnstructured(componentsregistry.NewLocalRepository(baseURL))
	return &lsv1alpha1.ComponentDescriptorDefinition{
		Reference: &lsv1alpha1.ComponentDescriptorReference{
			RepositoryContext: &repoCtx,
			ComponentName:     componentName,
			Version:           version,
		},
	}
}

// LocalRemoteBlueprintRef creates a new default local remote blueprint reference
func LocalRemoteBlueprintRef(resourceName string) lsv1alpha1.BlueprintDefinition {
	return lsv1alpha1.BlueprintDefinition{
		Reference: &lsv1alpha1.RemoteBlueprintReference{
			ResourceName: resourceName,
		},
	}
}

// ReadResourceFromFile reads a file and parses it to the given object
func ReadResourceFromFile(obj runtime.Object, testfile string) error {
	data, err := ioutil.ReadFile(testfile)
	if err != nil {
		return err
	}
	if _, _, err := api.Decoder.Decode(data, nil, obj); err != nil {
		return err
	}
	return nil
}

// ReadBlueprintFromFile reads a file and parses it to a Blueprint
func ReadBlueprintFromFile(testfile string) (*lsv1alpha1.Blueprint, error) {
	data, err := ioutil.ReadFile(testfile)
	if err != nil {
		return nil, err
	}
	blueprint := &lsv1alpha1.Blueprint{}
	if _, _, err := api.Decoder.Decode(data, nil, blueprint); err != nil {
		return nil, err
	}
	return blueprint, nil
}

// CreateBlueprintFromFile reads a blueprint from the given file and creates a internal blueprint object.
func CreateBlueprintFromFile(filePath, contentPath string) *blueprints.Blueprint {
	def, err := ReadBlueprintFromFile(filePath)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	contentPath, err = filepath.Abs(contentPath)
	gomega.Expect(err).To(gomega.Succeed())

	fs, err := projectionfs.New(osfs.New(), contentPath)
	gomega.Expect(err).To(gomega.Succeed())
	return blueprints.New(def, fs)
}

// CreateOrUpdateTarget creates or updates a target with specific name, namespace and type
func CreateOrUpdateTarget(ctx context.Context, client client.Client, namespace, name, ttype string, config interface{}) (*lsv1alpha1.Target, error) {
	data, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	target := &lsv1alpha1.Target{}
	target.Name = name
	target.Namespace = namespace

	_, err = controllerutil.CreateOrUpdate(ctx, client, target, func() error {
		target.Spec.Type = lsv1alpha1.TargetType(ttype)
		target.Spec.Configuration = lsv1alpha1.NewAnyJSON(data)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return target, err
}

// CreateKubernetesTarget creates a new target of type kubernetes
func CreateKubernetesTarget(namespace, name string, restConfig *rest.Config) (*lsv1alpha1.Target, error) {
	data, err := kutil.GenerateKubeconfigBytes(restConfig)
	if err != nil {
		return nil, err
	}

	config := lsv1alpha1.KubernetesClusterTargetConfig{
		Kubeconfig: lsv1alpha1.ValueRef{
			StrVal: pointer.StringPtr(string(data)),
		},
	}
	data, err = json.Marshal(config)
	if err != nil {
		return nil, err
	}

	target := &lsv1alpha1.Target{}
	target.Name = name
	target.Namespace = namespace

	target.Spec.Type = lsv1alpha1.KubernetesClusterTargetType
	target.Spec.Configuration = lsv1alpha1.NewAnyJSON(data)

	return target, nil
}

// CreateKubernetesTargetFromSecret creates a new target of type kubernetes from a secret
func CreateKubernetesTargetFromSecret(namespace, name string, secret *corev1.Secret) (*lsv1alpha1.Target, error) {
	// guess the key by using the first one
	var key string
	for sKey := range secret.Data {
		key = sKey
		break
	}

	config := lsv1alpha1.KubernetesClusterTargetConfig{
		Kubeconfig: lsv1alpha1.ValueRef{
			SecretRef: &lsv1alpha1.SecretReference{
				ObjectReference: lsv1alpha1.ObjectReference{
					Name:      secret.Name,
					Namespace: secret.Namespace,
				},
				Key: key,
			},
		},
	}
	data, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	target := &lsv1alpha1.Target{}
	target.Name = name
	target.Namespace = namespace

	target.Spec.Type = lsv1alpha1.KubernetesClusterTargetType
	target.Spec.Configuration = lsv1alpha1.NewAnyJSON(data)

	return target, nil
}

// BuildInternalKubernetesTarget creates a new target of type kubernetes
// whereas the hostname of the cluster will be set to the cluster internal host.
// It is expected that the controller runs inside the same cluster where it also deploys to.
func BuildInternalKubernetesTarget(ctx context.Context, kubeClient client.Client, namespace, name string, restConfig *rest.Config, internal bool) (*lsv1alpha1.Target, error) {
	if internal {
		oldHost := restConfig.Host
		defer func() {
			restConfig.Host = oldHost
		}()

		// get the kubernetes internal port of the kubernetes svc
		// it is expected that the kubernetes svc is in the default namespace with the name "kubernetes"
		const (
			kubernetesSvcName      = "kubernetes"
			kubernetesSvcNamespace = "default"
		)
		svc := &corev1.Service{}
		if err := kubeClient.Get(ctx, kutil.ObjectKey(kubernetesSvcName, kubernetesSvcNamespace), svc); err != nil {
			return nil, err
		}
		if len(svc.Spec.Ports) != 1 {
			return nil, fmt.Errorf("unexpected number of ports of the kubernetes service %d", len(svc.Spec.Ports))
		}
		u, err := url.Parse(oldHost)
		if err != nil {
			return nil, err
		}
		u.Host = fmt.Sprintf("%s.%s:%d", kubernetesSvcName, kubernetesSvcNamespace, svc.Spec.Ports[0].Port)
		restConfig.Host = u.String()
	}
	return lsutils.CreateKubernetesTarget(namespace, name, restConfig)
}

// BuildContainerDeployItem builds a new deploy item of type container.
func BuildContainerDeployItem(configuration *containerv1alpha1.ProviderConfiguration) *lsv1alpha1.DeployItem {
	di, err := container.NewDeployItemBuilder().
		ProviderConfig(configuration).
		Build()
	ExpectNoErrorWithOffset(1, err)
	return di
}

// ReadAndCreateOrUpdateDeployItem reads a deploy item from the given file and creates or updated the deploy item
func ReadAndCreateOrUpdateDeployItem(ctx context.Context, testenv *envtest.Environment, state *envtest.State, diName, file string) *lsv1alpha1.DeployItem {
	kubeconfigBytes, err := kutil.GenerateKubeconfigJSONBytes(testenv.Env.Config)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	di := &lsv1alpha1.DeployItem{}
	ExpectNoError(ReadResourceFromFile(di, file))
	di.Name = diName
	di.Namespace = state.Namespace
	di.Spec.Target = &lsv1alpha1.ObjectReference{
		Name:      "test-target",
		Namespace: state.Namespace,
	}

	// Create Target
	target, err := CreateOrUpdateTarget(ctx,
		testenv.Client,
		di.Spec.Target.Namespace,
		di.Spec.Target.Name,
		string(lsv1alpha1.KubernetesClusterTargetType),
		lsv1alpha1.KubernetesClusterTargetConfig{
			Kubeconfig: lsv1alpha1.ValueRef{
				StrVal: pointer.StringPtr(string(kubeconfigBytes)),
			},
		},
	)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	gomega.Expect(state.AddResources(target)).To(gomega.Succeed())

	old := &lsv1alpha1.DeployItem{}
	if err := testenv.Client.Get(ctx, kutil.ObjectKey(di.Name, di.Namespace), old); err != nil {
		if apierrors.IsNotFound(err) {
			gomega.Expect(state.Create(ctx, di, envtest.UpdateStatus(true))).To(gomega.Succeed())
			return di
		}
		ExpectNoError(err)
	}
	di.ObjectMeta = old.ObjectMeta
	ExpectNoError(testenv.Client.Patch(ctx, di, client.MergeFrom(old)))
	return di
}
