# Helm Deployer

The helm deployer is a controller that reconciles DeployItems of type `landscaper.gardener.cloud/helm`. It renders a given helm chart and deploys the resulting manifest into a cluster.

It also checks by default the health of the deployed resources. See [healthchecks.md](healthchecks.md) for more info.

**Index**:
- [Provider Configuration](#provider-configuration)
- [Provider Status](#status)
- [Deployer Configuration](#deployer-configuration)

### Provider Configuration

This sections describes the provider specific configuration.

```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: DeployItem
metadata:
  name: my-nginx
spec:
  type: landscaper.gardener.cloud/helm
  
  target: # has to be of type landscaper.gardener.cloud/kubernetes-cluster
    name: my-cluster
    namespace: test

  config:
    apiVersion: helm.deployer.landscaper.gardener.cloud/v1alpha1
    kind: ProviderConfiguration
    
    chart:
      ref: myrepo.example.com/charts/nginx-ingress:0.5.2 # helm oci ref
      fromResource: # will fetch the helm chart from component descriptor resource of type helm chart
#       inline: # define an inline component descriptor instead of referencing a remote
        ref:
          repositoryContext:
            type: ociRegistry
            baseUrl: my-repo
          componentName: github.com/gardener/landscaper
          version: v0.3.0
        resourceName: my-helm-chart
      archive:
        raw: "" # base64 encoded helm chart tar.gz
        remote:
          url: "https://my-domain/maychart.tar.gz"

    # base64 encoded kubeconfig pointing to the cluster to install the chart
    kubeconfig: xxx

    updateStrategy: update | patch # optional; defaults to update

#    continuousReconcile: # configure continuous reconciliation
#      every: "1h"
#      cron: cron: "* */1 * * *"

    # Configuration of the readiness checks for the resources.
    # optional
    readinessChecks:
      # Allows to disable the default readiness checks.
      # optional; set to false by default.
      disableDefault: true
      # Defines the time to wait before giving up on a resource
      # to be ready. Should be changed with long startup time pods.
      # optional; default to 180 seconds/3 minutes.
      timeout: 3m
      # Configuration of custom readiness checks which are used
      # to check on custom fields and their values
      # especially useful for resources that came in through CRDs
      # optional
      custom:
      # the name of the custom readiness check, required
      - name: myCustomReadinessCheck
        # timeout of the custom readiness check
        # optional, defaults to the timeout stated above
        timeout: 2m
        # temporarily disable this custom readiness check, useful for test setups
        # optional, defaults to false
        disabled: false
        # a specific resource should be selected for this readiness check to be performed on
        # a resource is uniquely defined by its GVK, namespace and name
        # required if no labelSelector is specified, can be combined with a labelSelector which is potentially harmful
        resourceSelector:
          apiVersion: apps/v1
          kind: Deployment
          name: myDeployment
          namespace: myNamespace
        # multiple resources for the readiness check to be performed on can be selected through labels
        # they are identified by their GVK and a set of labels that all need to match
        # required if no resourceSelector is specified, can be combined with a resourceSelector which is potentially harmful
        labelSelector:
          apiVersion: apps/v1
          kind: Deployment
          matchLabels:
            app: myApp
            component: backendService
        # requirements specifies what condition must hold true for the given objects to pass the readiness check
        # multiple requirements can be given and they all need to successfully evaluate
        requirements:
        # jsonPath denotes the path of the field of the selected object to be checked and compared
        - jsonPath: .status.readyReplicas
          # operator specifies how the contents of the given field should be compared to the desired value
          # allowed operators are: DoesNotExist(!), Exists(exists), Equals(=, ==), NotEquals(!=), In(in), NotIn(notIn)
          operator: In
          # values is a list of values that the field at jsonPath must match to according to the operators
          values:
          - value: 1
          - value: 2
          - value: 3

    # Defines the time to wait before giving up on a resource to be deleted,
    # for instance when deleting resources that are not anymore managed from this DeployItem.
    # optional; default to 180 seconds/3 minutes.
    deleteTimeout: 2m

    # Name of the release: helm install [name]
    name: my-release
    # Namespace of the release: helm --namespace
    namespace: default
    # configure the landscaper to automatically create the release namespace.
    # Works similar to "helm install --create-namespace"
    createNamespace: true
    # Values to template the chart
    # optional
    values: {}

    # Define exports that are read from the manifests and exported so 
    # that is can be used by other deployitems.
    # The exports are read form the deployed resources so status and other runtime attributes
    # are available and can be exported.
    exports:
      defaultTimeout: 5m
      exports:
      # Describes one export that is read from the templates values or a templated resource.
      # The value will be by default read from the values if fromResource is not specified.
      # The specified jsonPath value is written with the given key to the exported configuration.
      - key: KeyA # value is read from the values file
        jsonPath: .Values.keyA
      - key: KeyB # value is read from secret
        timeout: 10m # optional specific timeout
        jsonPath: .spec.config
        fromResource:
          apiVersion: v1
          kind: Secret
          name: my-secret
          namespace: a
      - key: KeyC # value is read from secret that is referenced by a service account
        timeout: 10m # optional specific timeout
        jsonPath: .secrets[0] # points to a object ref that consists of a name and namespace
        fromResource:
          apiVersion: v1
          kind: ServiceAccount
          name: my-user
          namespace: a
        # Defines the referenced objects kind and version. 
        # The name and namespace is taken from the resource defined in "fromResource".
        fromObjectRef: 
          apiVersion: v1
          kind: Secret
          jsonPath: ".data.somekey" # jsonpath in the secret

    
    exportsFromManifests: [] # same as exports.exports but deprecated
```

Exports can be defined in `exportsFromManifests` by specifying the exported key to export.
The value is taken from a rendered resource and a jsonpath to the value.
For a complete documention of the availabel jsonPath see here (https://kubernetes.io/docs/reference/kubectl/jsonpath/).

:warning: Only unique identifiable resources (_apiVersion_, _kind_, _name_ and _namespace_).

##### Continuous Reconciliation
For information on the continuous reconciliation configuration, see [here](../development/deployer-extensions##continuous-reconcile-extension) under 'usage'.

### Status

This section describes the provider specific status of the resource.

```yaml
status:
  providerStatus:
    apiVersion: helm.deployer.landscaper.gardener.cloud
    kind: ProviderStatus
    managedResources:
    - apiGroup: k8s.apigroup.com/v1
      kind: my-type
      name: my-resource
      namespace: default
```

## Deployer Configuration

When deploying the helm deployer controller it can be configured using the `--config` flag and providing a configuration file.

The structure of the provided configuration file is defined as follows.

:warning: Keep in mind that when deploying with the helm chart the configuration is abstracted using the helm values. See the [helm values file](../../charts/helm-deployer/values.yaml) for details when deploying with the helm chart.
```yaml
apiVersion: helm.deployer.landscaper.gardener.cloud/v1alpha1
kind: Configuration
oci:
  # allow plain http connections to the oci registry.
  # Use with care as the default docker registry does not serve http with any authentication
  allowPlainHttp: false
  # skip the tls validation
  insecureSkipVerify: false
  # path to docker compatible auth configuration files.
#  configFiles:
#  - "somepath"

# target selector to only react on specific deploy items.
# see the common config in "./README.md" for detailed documentation.
targetSelector:
  annotations: []
  labels: []
```
