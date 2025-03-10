{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "deployer.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "deployer.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "deployer.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "deployer.labels" -}}
helm.sh/chart: {{ include "deployer.chart" . }}
{{ include "deployer.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "deployer.selectorLabels" -}}
app.kubernetes.io/name: {{ include "deployer.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "deployer.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "deployer.fullname" .) .Values.serviceAccount.name }}-tmp
{{- else }}
{{- default "default" .Values.serviceAccount.name }}-tmp
{{- end }}
{{- end }}

{{/*
Creates the container deployer configuration file which will be encapsulated in a secret.
*/}}
{{- define "deployer-config" -}}
apiVersion: container.deployer.landscaper.gardener.cloud/v1alpha1
kind: Configuration
{{- if .Values.deployer.identity }}
identity: {{ .Values.deployer.identity }}
{{- end }}
namespace: {{ .Values.deployer.namespace | default .Release.Namespace  }}
initContainer:
  image: "{{ include "init-image" . }}"
waitContainer:
  image: "{{ include "wait-image" . }}"
{{- if .Values.deployer.defaultImage }}
defaultImage:
  image: "{{ include "utils-templates.image" .Values.deployer.defaultImage }}"
{{- end }}
{{- if .Values.deployer.oci }}
oci:
  allowPlainHttp: {{ .Values.deployer.oci.allowPlainHttp }}
  insecureSkipVerify: {{ .Values.deployer.oci.insecureSkipVerify }}
  {{- if .Values.deployer.oci.secrets }}
  configFiles:
  {{- range $key, $value := .Values.deployer.oci.secrets }}
  - /app/ls/registry/secrets/{{ $key }}
  {{- end }}
  {{- end }}
{{- end }}
{{- with .Values.deployer.targetSelector }}
targetSelector:
{{ toYaml . }}
{{- end }}
{{- end }}

{{- define "deployer-image" -}}
{{- $tag := ( .Values.image.tag | default .Chart.AppVersion )  -}}
{{- $image :=  dict "repository" .Values.image.repository "tag" $tag  -}}
{{- include "utils-templates.image" $image }}
{{- end -}}

{{- define "init-image" -}}
{{- $tag := ( .Values.deployer.initContainer.tag | default .Chart.AppVersion )  -}}
{{- $image :=  dict "repository" .Values.deployer.initContainer.repository "tag" $tag  -}}
{{- include "utils-templates.image" $image }}
{{- end -}}

{{- define "wait-image" -}}
{{- $tag := ( .Values.deployer.waitContainer.tag | default .Chart.AppVersion )  -}}
{{- $image :=  dict "repository" .Values.deployer.waitContainer.repository "tag" $tag  -}}
{{- include "utils-templates.image" $image }}
{{- end -}}

{{- define "utils-templates.image" -}}
{{- if hasPrefix "sha256:" (required "$.tag is required" $.tag) -}}
{{ required "$.repository is required" $.repository }}@{{ required "$.tag is required" $.tag }}
{{- else -}}
{{ required "$.repository is required" $.repository }}:{{ required "$.tag is required" $.tag }}
{{- end -}}
{{- end -}}
