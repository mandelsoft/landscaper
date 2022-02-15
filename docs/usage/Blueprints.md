# Blueprints

A Blueprint is a parameterized description of how to deploy a specific component.

The description follows the kubernetes operator approach:

The task of a Blueprint is to provide deployitem descriptions based on its input and outputs based on the input and the state of the deployment.

The rendered deployitems are then handled by independent kubernetes operators, which perform the real deployment tasks. This way, the Blueprint does not execute deployment actions, but provides the target state of formally described deployitems. The actions described by the Blueprint itself are therefore restricted to YAML-based manifest rendering. These actions are described by [template executions](./Templating.md).

A Blueprint is a filesystem structure that contains the blueprint definition at `/blueprint.yaml`. Any other additional file can be referred to in the blueprint.yaml for JSON schema definitions and templates.

Every Blueprint must have a corresponding component descriptor that is used to reference the Blueprint and define its the dependencies.

```
my-blueprint
├── data
│   └── myadditional data
└── blueprint.yaml
```

The blueprint definition (blueprint.yaml) describes
- declaration of import parameters
- declaration of export parameters
- JSONSchema definitions
- generation rules for deployitems
- generation rules for export values
- generation of nested installations


**Index**:
- [Blueprints](#blueprints)
  - [Example](#blueprintyaml-definition)
  - [Import Definitions](#import-definitions)
  - [Export Definitions](#export-definitions)
  - [JSONSchema](#jsonschema)
  - [Rendering](#rendering)
    - [DeployItems](#deployitems)
    - [Export Values](#export-values)
    - [Nested Installations](#nested-installations)
  <!-- - [Remote Access](#remote-access)
    - [Local](#local)
    - [OCI](#oci) -->

## Example

The following snippet shows the structure of a `blueprint.yaml` file. It is expected as top-level file in the blueprint filesystem structure. Refer to [apis/.schemes/core-v1alpha1-Blueprint.json](../../apis/.schemes/core-v1alpha1-Blueprint.json) for the automatically generated jsonschema definition.

```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Blueprint

# jsonSchemaVersion describes the default jsonschema definition 
# for the import and export definitions.
jsonSchemaVersion: "https://json-schema.org/draft/2019-09/schema"

# localTypes defines shared jsonschema types that can be used in the 
# import and export definitions.
localTypes:
  mytype: # expects a jsonschema
    type: object

# imports defines all imported values that are expected.
# Data can be either imported as data object or target.
imports:
# Import a data object by specifying the expected structure of data
# as jsonschema.
- name: my-import-key
  required: true # required, defaults to true
  type: data # this is a data import
  schema: # expects a jsonschema
    "$ref": "local://mytype" # references local type
# Import a target by specifying the targetType
- name: my-target-import-key
  required: true # required, defaults to true
  type: target # this is a target import
  targetType: landscaper.gardener.cloud/kubernetes-cluster
# Import a targetlist
- name: my-targetlist-import-key
  # required: true # defaults to true
  type: targetList # this is a targetlist import
  targetType: landscaper.gardener.cloud/kubernetes-cluster
# Import a component descriptor
- name: my-cd-import-key
  # required: true # defaults to true
  type: componentDescriptor # this is a component descriptor import
# Import a component descriptor list
- name: my-cdlist-import-key
  # required: true # defaults to true
  type: componentDescriptorList # this is a component descriptor list import

# exports defines all values that are produced by the blueprint
# and that are exported.
# Exported values can be consumed by other blueprints.
# Data can be either exported as data object or target.
exports:
# Export a data object by specifying the expected structure of data
# as jsonschema.
- name: my-export-key
  type: data # this is a data export
  schema: # expects a jsonschema
    type: string
# Export a target by specifying the targetType
- name: my-target-export-key
  type: target # this is a target export
  targetType: landscaper.gardener.cloud/kubernetes-cluster

# deployExecutions are a templating mechanism to 
# template the deploy items.
# For detailed documentation see #DeployExecutions
deployExecutions: 
- name: execution-name
  type: GoTemplate
  file: <path to file> # path is relative to the blueprint's filesystem root

# exportExecutions are a templating mechanism to 
# template the export.
# For detailed documentation see #ExportExecutions
exportExecutions:
- name: execution-name
  type: Spiff
  template: # inline template

# subinstallations is a list of installation templates.
# A installation template expose specific installation configuration are 
# used to assemble multiple blueprints together.
subinstallations:
- file: /installations/dns/dns-installation.yaml
- apiVersion: landscaper.gardener.cloud/v1alpha1
  kind: InstallationTemplate
  name: ingress # must be unique
  blueprint:
    ref: cd://componentReferences/ingress/resources/blueprint #cd://resources/myblueprint
#    filesystem:
#      blueprint.yaml: abc...
  
  # define imported dataobjects and target from other installations or the 
  # parents import.
  # It's the same syntax as for default installations.
  imports:
    data:
    - name: "parent-data" # data import name
      dataRef: "data" # dataobject name - refers to import of parent
    targets:
    - name: "" # target import name
      target: "" # target name
  #importMappings: {}

  exports:
    targets:
    - name: "" # target export name
      target: "" # target name
  #exportMappings: {}

```

### Import Definitions

Blueprints describe formal imports. A formal import parameter has a name and a *value type*. It may describe a single simple value or a complex data structure. There are several *types* of imports, indicating different use cases:
- **`data`**
  This type of import is used to import arbitrary data according to its value type. The value type is described by a [JSONSchema](#jsonschema).
- **`target`**
  This type declares an import of a [deployment target object](./Targets.md). It is used in the rendered deployitems to specify the target environment for the deployment of the deployitem.
- **`targetList`**
  This type can be used if, instead of a single target object, an arbitrary number of targets should be imported. All targets imported as part of a targetlist import must have the same `targetType`.
- **`componentDescriptor`**
  This type refers to an import of a component descriptor.
- **`componentDescriptorList`**
  Analogous to `targetList`, this type allows importing an arbitrary number of component descriptors.

The imports are described as a list of import declarations in the blueprint field `.imports`. An import declaration has the following fields:
- **`name`** *string*
  Identfier for the import parameter. Can be used in the templating to access the actual import value provided by the installation.
- **`type`** *type*
  The type of the import as described above.
  For backward compatibility, the `type` field is currently optional for *data* and *target* imports, but it is strongly recommended to specify it for each import declaration.
- **`required`** *bool* (default: `true`)
  If set to false, the installation does not have to provide this import.
- **`default`** *any*
  If the import is not required and not provided by the installation, this default value will be used for it.
- **`imports`** *list of import declarations*
  Nested imports only exist if the owning import is satisfied. Cannot be specified for a required import. See [here](./ConditionalImports.md) for further details.
- **`schema`** *JSONSchema*
  Must be set for imports of type `data` (only). Describes the structure of the expected import value as [JSONSchema](#jsonschema).
- **`targetType`** *string*
  Must be set for imports of type `target` and `targetList` (only). It declares the type of the expected [*Target*](./Targets.md) object. If the `targetType` does not contain a `/`, it will be prefixed with `landscaper.gardener.cloud/`.

**Example**
```yaml
imports:
- name: myimport # some unique name
  required: false # defaults to true if not set
  type: data # type of the imported object
  schema:
    type: object
    properties:
      username:
        type: string
      password:
        type: string
  default:
    username: foo
    password: bar
- name: mycluster
  type: target
  targetType: kubernetes-cluster # will be defaulted to 'landscaper.gardener.cloud/kubernetes-cluster'
```

## Export Definitions

Blueprints describe formal exports. The export declarations are very similar to the import declarations. The following types can be exported:
- **`data`**
  This type of export is used to export arbitrary data according to its value type. The value type is described by a [JSONSchema](#jsonschema).
- **`target`**
  This type declares an export of a [deployment target object](./Targets.md). It is used in the rendered deployitems to specify the target environment for the deployment of the deployitem.

The exports are described as a list of export declarations in the blueprint field `.exports`. An export declaration has the following fields:
- **`name`** *string*
  Identfier for the export parameter. Can be used in the templating to access the actual export value provided by the installation.
- **`type`** *type*
  The type of the export as described above.
  For backward compatibility, the `type` field is currently optional for *data* and *target* exports, but it is strongly recommended to specify it for each export declaration.
- **`schema`** *JSONSchema*
  Must be set for exports of type `data` (only). Describes the structure of the expected export value as [JSONSchema](#jsonschema).
- **`targetType`** *string*
  Must be set for exports of type `target` (only). It declares the type of the expected [*Target*](./Targets.md) object. If the `targetType` does not contain a `/`, it will be prefixed with `landscaper.gardener.cloud/`.

**Example**
```yaml
exports:
- name: myexport
  type: data
  schema:
    type: object
    properties:
      username:
        type: string
      password:
        type: string
- name: myclusterexport
  type: target
  targetType: kubernetes-cluster # will be defaulted to 'landscaper.gardener.cloud/kubernetes-cluster'
```


## JSONSchema

[JSONSchemas](https://json-schema.org/) are used to describe the structure of `data` imports and exports. The provided import schema is used to validate the actual import value before executing the blueprint.

It is recommended to provide a description and an example for the structure, so that users of the blueprint know what to provide (see the [json docs](http://json-schema.org/understanding-json-schema/reference/generic.html#annotations)).

For detailed information about the jsonschema and landscaper specifics see [JSONSchema Docs](./JSONSchema.md)

### Templating

All template executions get a common standardized binding:
- **`imports`**
  the imports of the installation, as a mapping from import name to assigned values
- **`cd`**
  the component descriptor of the owning component
- **`blueprintDef`**
  the blueprint definition, as given in the installation (not the blueprint.yaml itself)
- **`componentDescriptorDef`**
  the component descriptor definition, as given in the installation (not the component descriptor itself)

#### DeployItem Templates

A Blueprint's deploy executions may contain any number of template executors. 
A template executor must return a list of deploy items templates.<br>
A deploy item template exposes specific deploy item fields and will be rendered to DeployItem CRs by the landscaper.

__DeployItem Template__:
```yaml
deployItems:
- name: deploy-item-name # unique identifier of the step
  target:
    name: ""
    namespace: ""
  config:
    apiVersion: mydeployer/v1
    kind: ProviderConfiguration
    ...
```

##### Executor Imports

All template executors are given the same input data that can be used while templating.
The input consists of the imported values as well as the installations component descriptor.

For the specific documentation about the available templating engines see [Template Executors](./TemplateExecutors.md).

```yaml
imports:
  <import-name>: <import value>
cd: <component descriptor>
components: <list of all referenced component descriptors>
blueprint: <blueprint definition> # blueprint definition from the Installation
componentDescriptorDef: <component descriptor definition> # component descriptor definition from the installation
```

All list of deployitem templates of all template executors are appended to one list as they are specified in the deployExecution.

*Example*:

Input values:
```yaml
imports:
  replicas: 3
  cluster:
    apiVersion: landscaper.gardener.cloud/v1alpha1
    kind: Target
    metadata:
       name: dev-cluster
       namespace: default
    spec:
      type: landscaper.gardener.cloud/kubernetes-cluster
      config:
        kubeconfig: |
          apiVersion: ...
  my-cdlist: # import of a component descriptor list
    meta:
      schemaVersion: v2
    components:
      - meta:
          schemaVersion: v2
        component: 
          name: component-1
          version: v1.0.1
          ...  # same structure as for key "cd"   
      - meta:
          schemaVersion: v2
        component:
          name: component-2
          version: v1.0.1
          ...
cd:
  meta:
    schemaVersion: v2
  component:
    name: my-component
    version: v1.0.0
    componentReferences:
    - name: abc
      componentName: my-referenced-component
      version: v1.0.0
    resources:
    - name: nginx-ingress-chart
      version: 0.30.0
      relation: external
      acccess:
        type: ociRegistry
        imageReference: nginx:0.30.0
components:
- meta: # the resolved component referenced in "cd.component.componentReferences[0]"
    schemaVersion: v2
  component:
    name: my-referenced-component
    version: v1.0.0
    resources:
    - name: ubuntu
      version: 0.18.0
      relation: external
      acccess:
        type: ociRegistry
        imageReference: ubuntu:0.18.0
blueprint:
 ref:
  #      repositoryContext:
  #        type: ociRegistry
  #        baseUrl: eu.gcr.io/myproj
  componentName: github.com/gardener/gardener
  version: v1.7.2
  resourceName: gardener
#    inline:
#      filesystem: # vfs filesystem
#        blueprint.yaml: 
#          apiVersion: landscaper.gardener.cloud/v1alpha1
#          kind: Blueprint
#          ...
```


```yaml
deployExecutions:
- name: default
  type: GoTemplate
  template: |
    deployItems:
    - name: deploy
      type: landscaper.gardener.cloud/helm
      target:
        name: {{ .imports.cluster.metadata.name }} # will resolve to "dev-cluster"
        namespace: {{ .imports.cluster.metadata.namespace  }} # will resolve to "default"
      config:
        apiVersion: helm.deployer.landscaper.gardener.cloud/v1alpha1
        kind: ProviderConfiguration
        
        chart:
          {{ $resource := getResource .cd "name" "nginx-ingress-chart" }}
          ref: {{ $resource.access.imageReference }} # resolves to nginx:0.30.0
        
        values:
          replicas: {{ .imports.replicas  }} # will resolve to 3
          
          {{ $component := getComponent .cd "name" "my-referenced-component" }} # get a component that is referenced
          {{ $resource := getResource $component "name" "ubuntu" }}
          usesImage: {{ $resource.access.imageReference }} # resolves to ubuntu:0.18.0
          
          imageVectorOverwrite: |
            {{- generateImageOverwrite .cd .imports.my-cdlist | toYaml | nindent 12 }}
```

#### Export Templates

A Blueprint's export executions may contain any number of template executors. 
A template executor must return a map of `export name` to `exported value`.<br>
Multiple template executor exports will be merged in the defined order, whereas the later defined values overwrites previous templates.

__exports__:
```yaml
exports:
  export-name: export-value
  target-export-name:
    labels: {}
    annotations: {}
    type: "" # target type
    config: any # target specific config data
```

All template executors are given the same input data that can be used while templating.
The input consists of the deploy items export values and all exports of subinstallations.

For the specific documentation about the available templating engines see [Template Executors](./TemplateExecutors.md).

```yaml
values:
  deployitems:
    <deployitem step name>: <exported values>
  dataobjects:
      <databject name>: <data object value>
  targets:
        <target name>: <whole target>
```

All list of deployitem templates of all template executors are appended to one list as they are specified in the deployExecution.

*Example*:

Input values:
```yaml
values:
  deployitems:
    deploy:
      ingressPrefix: my-pref
  dataobjects:
     domain: example.com
  targets:
    dev-cluster:
        apiVersion: landscaper.gardener.cloud/v1alpha1
        kind: Target
        metadata:
           name: dev-cluster
           namespace: default
        spec:
          type: landscaper.gardener.cloud/kubernetes-cluster
          config:
            kubeconfig: |
              apiVersion: ...
```

```yaml
exportExecutions:
- name: default
  type: GoTemplate
  template: |
    exports:
      url: http://{{ .values.deployitems.ingressPrefix  }}.{{ .values.dataobjects.domain }} # resolves to http://my-pref.example.com
      cluster:
        type: {{ .values.targets.dev-cluster.spec.type  }}
        config: {{ .values.targets.dev-cluster.spec.config  }}
```

#### Installation Templates
Installation Templates are used to include subinstallation in a blueprint.
As the name suggest, they are templates for installation which means that the landscaper will create installation based on these templates.

These subinstallations have a context that is defined by the parent installation.
Context means that subinstallations can only import data that is also imported by the parent or exported by other subinstallations with the same parent.

Installation templates offer the same configuration as real installation 
except that blueprints have to be defined in the component descriptor of the blueprint (either as resource or by a component reference).
Inline blueprints are also possible.

Subinstallations can also be defined in a separate file or templated via executor (templated executors are defined in a separate field `.subinstallationExecutions`).
If defined by file it is expected that that file contains one InstallationTemplate.

All possible options to define a subinstallation can be used in parallel and are summed up.

```yaml
- apiVersion: landscaper.gardener.cloud/v1alpha1
  kind: InstallationTemplate
  name: my-subinstallation # must be unique
  blueprint:
    ref: cd://componentReferences/ingress/resources/blueprint #cd://resources/myblueprint
#    filesystem:
#      blueprint.yaml: abc...
  
  # define imported dataobjects and target from other installations or the 
  # parents import.
  # It's the same syntax as for default installations.
  imports:
    data:
    - name: "" # data import name
      dataRef: "" # dataobject name
    targets:
    - name: "" # target import name
      target: "" # target name
    - name: ""
      targetListRef: "" # references a targetlist import of the parent
    componentDescriptors:
    - name: ""
      dataRef: "" # references a component descriptor (single or list) import of the parent
  #importMappings: {}

  exports:
    targets:
    - name: "" # target export name
      target: "" # target name
  #exportMappings: {}
```

Similar to how deploy items can be defined, it is also possible to create template subinstallations based on the imports.
A Blueprint's subinstallations executions may contain any number of template executors.
A template executor must return a list of installation templates.<br>

For a list of available templating imports see the [deploy item executor docs](#executor-imports).

__Subinstallation Template__:
```yaml
subinstallationExecutions:
- name: default
  type: GoTemplate
  template: |
    subinstallations:
    - apiVersion: landscaper.gardener.cloud/v1alpha1
      kind: InstallationTemplate
      name: my-subinstallation # must be unique
      blueprint:
        ref: cd://componentReferences/ingress/resources/blueprint
      ...
```


##### Targetlist Imports in Subinstallations

To import a targetlist that has been imported by a parent installation, use `targetListRef` to reference the name of the parent import.

```yaml
imports:
  targets:
  - name: "my-foo-targets"
    targets: # targetlist import
    - foo
    - foobar
    - foobaz
subinstallations:
- apiVersion: landscaper.gardener.cloud/v1alpha1
  kind: InstallationTemplate
  name: mysubinst
  imports:
    targets:
    - name: "also-my-foo-targets"
      targetListRef: "my-foo-targets"
```


##### Component Descriptor Imports in Subinstallations

Only root installations can directly reference component descriptors in their imports. In subinstallations, it is only possible to reference a component descriptor which has already been imported by the parent. Therefore, only the fields `dataRef` and `list` are allowed in component descriptor imports in subinstallations.
With `dataRef`, a single component descriptor or a list of component descriptors imported by the parent can be referenced.
The `list` field can be used to build a new component descriptor list import, in the same way it is used in regular installations. The only difference is that all list entries can only use `dataRef` to reference a component descriptor.

```yaml
imports:
  componentDescriptors:
  - name: "my-single-cd"
    secretRef: ... # single component descriptor import
  - name: "my-other-single-cd"
    secretRef: ... # single component descriptor import
  - name: "my-cd-list"
    list: # component descriptor list import
    - secretRef: ...
    - configMapRef: ...
subinstallations:
- apiVersion: landscaper.gardener.cloud/v1alpha1
  kind: InstallationTemplate
  name: mysubinst
  imports:
    componentDescriptors:
    - name: "also-my-single-cd" # single component descriptor reference
      dataRef: "my-single-cd"
    - name: "also-my-cd-list" # component descriptor list reference
      dataRef: "my-cd-list"
    - name: "new-cd-list" # a new component descriptor list import based on multiple single cd imports
      list:
      - dataRef: "my-single-cd"
      - dataRef: "my-other-single-cd"
```


## Remote Access

Blueprints are referenced in installations or installation templates via the component descriptors access.

Basically blueprints are a filesystem, therefore, any blob store could be supported.<br>
Currently, local and OCI registry access is supported.

:warning: Be aware that a local reigstry should be only used for testing and development, whereas the OCI registry is the preferred productive method.


### Local

A local registry can be defined in the landscaper configuration by providing the below configuration.
The landscaper expects the given paths to be a directory that contains the definitions in subdirectory.
The subdirectory should contain the file `description.yaml`, that contains the actual ComponentDefinition with its version and name.
The whole subdirectory is used as the blob content of the Component.
```yaml
apiVersion: config.landscaper.gardener.cloud/v1alpha1
kind: LandscaperConfiguration

registry:
  local:
    paths:
    - "/path/to/definitions"
```

The blueprints are referenced via `local` access type in the component descriptor.
```
component:
  localResource:
  - name: blueprint
    type: blueprint
    access:
      type: local
```

### OCI

ComponentDefinitions can be stored in a OCI compliant registry which is the preferred way to create and offer ComponentDefinitions.
The Landscaper uses [OCI Artifacts](https://github.com/opencontainers/artifacts) which means that a OCI compliant registry has to be used.
For more information about the [OCI distribution spec](https://github.com/opencontainers/distribution-spec/blob/master/spec.md) and OCI compliant registries refer to the official documents.

The OCI manifest is stored in the below format in the registry.
Whereas the config is ignored and there must be exactly one layer with the containing a bluprints filesystem as `application/tar+gzip`.
 
 The layers can be identified via their title annotation or via their media type as only one component descriptor per layer is allowed.
```json
{
   "schemaVersion": 2,
   "annotations": {},
   "config": {},
   "layers": [
      {
         "digest": "sha256:6f4e69a5ff18d92e7315e3ee31c62165ebf25bfa05cad05c0d09d8f412dae401",
         "mediaType": "application/tar+gzip",
         "size": 78343,
         "annotations": {
            "org.opencontainers.image.title": "definition"
         }
      }
   ]
}
```

The blueprints are referenced via `ociRegistry` access type in the component descriptor.
```
component:
  localResource:
  - name: blueprint
    type: blueprint
    access:
      type: ociRegistry
      imgageReference: oci-ref:1.0.0
```
