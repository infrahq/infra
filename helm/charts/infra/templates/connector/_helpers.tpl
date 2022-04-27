{{/*
Expand the name of the chart.
*/}}
{{- define "connector.name" -}}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if hasSuffix .Values.connector.componentName  $name }}
{{- $name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" $name .Values.connector.componentName | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "connector.fullname" -}}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if .Values.fullnameOverride }}
{{- $name = .Values.fullnameOverride }}
{{- else }}
{{- if contains $name .Release.Name }}
{{- $name = .Release.Name }}
{{- else }}
{{- $name = printf "%s-%s" .Release.Name $name }}
{{- end }}
{{- end }}
{{- if hasSuffix .Values.connector.componentName $name }}
{{- $name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" $name .Values.connector.componentName | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "connector.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "connector.labels" -}}
helm.sh/chart: {{ include "connector.chart" . }}
app.infrahq.com/component: {{ .Values.connector.componentName }}
{{ include "connector.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- if or .Values.connector.labels .Values.global.labels }}
{{ merge .Values.connector.labels .Values.global.labels | toYaml }}
{{- end }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "connector.selectorLabels" -}}
app.kubernetes.io/name: {{ include "connector.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Pod labels
*/}}
{{- define "connector.podLabels" -}}
{{- include "connector.selectorLabels" . }}
{{- if or .Values.connector.podLabels .Values.global.podLabels }}
{{ merge .Values.connector.podLabels .Values.global.podLabels | toYaml }}
{{- end }}
{{- end }}

{{/*
Pod annotations
*/}}
{{- define "connector.podAnnotations" -}}
rollme: {{ include (print .Template.BasePath "/connector/configmap.yaml") . | sha1sum }}
{{- if or .Values.connector.podAnnotations .Values.global.podAnnotations }}
{{ merge .Values.connector.podAnnotations .Values.global.podAnnotations | toYaml }}
{{- end }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "connector.serviceAccountName" -}}
{{- if .Values.connector.serviceAccount.create }}
{{- default (include "connector.fullname" .) .Values.connector.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.connector.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Connector image repository.
If global value is present, use global value. Otherwise, use local value.
*/}}
{{- define "connector.image.repository" -}}
{{- if .Values.global.image }}
{{- .Values.global.image.repository | default .Values.connector.image.repository }}
{{- else }}
{{- .Values.connector.image.repository }}
{{- end }}
{{- end }}

{{/*
Connector image tag.
If a local override exists, use the local override. Otherwise, if a global
override exists, use the global override.  If `image.tag` does not exist,
use AppVersion defined in Chart.
*/}}
{{- define "connector.image.tag" -}}
{{- if .Values.global.image }}
{{- .Values.connector.image.tag | default .Values.global.image.tag | default .Chart.AppVersion }}
{{- else }}
{{- .Values.connector.image.tag | default .Chart.AppVersion }}
{{- end }}
{{- end }}

{{/*
Connector image pull policy.
If global value is present, use global value. Otherwise, use local value.
*/}}
{{- define "connector.image.pullPolicy" -}}
{{- if .Values.global.image }}
{{- .Values.global.image.pullPolicy | default .Values.connector.image.pullPolicy }}
{{- else }}
{{- .Values.connector.image.pullPolicy }}
{{- end }}
{{- end }}

{{/*
Connector image pull secrets. Merges global and local values.
*/}}
{{- define "connector.imagePullSecrets" -}}
{{ concat .Values.connector.imagePullSecrets .Values.global.imagePullSecrets | uniq | toYaml }}
{{- end }}

{{/*
Create an admin access key. If one is defined through values, use it. Otherwise look for an
existing secret and use its password. If the secret does not exist, randomly generate a password.
*/}}
{{- define "connector.accessKey" -}}
{{- if .Values.connector.config.accessKey }}
{{- .Values.connector.config.accessKey }}
{{- else }}
{{- $secret := lookup "v1" "Secret" .Release.Namespace (printf "%s-access-key" .Release.Name) }}
{{- if $secret }}
{{- index $secret "data" "access-key" | b64dec }}
{{- else }}
{{- randAlphaNum 10 }}.{{ randAlphaNum 24 }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Infra connector 'env' values. Merges global and local values.
*/}}
{{- define "connector.env" -}}
{{- concat .Values.connector.env .Values.global.env | uniq | toYaml }}
{{- end }}

{{/*
Infra connector 'envFrom' values. Merges global and local values.
*/}}
{{- define "connector.envFrom" -}}
{{- concat .Values.connector.envFrom .Values.global.envFrom | uniq | toYaml }}
{{- end }}

{{/*
Infer whether Infra connector should be deployed based on connector.enabled, connector.config.server, and connector.config.accessKey.
*/}}
{{- define "connector.enabled" -}}
{{- or .Values.connector.enabled (not (empty .Values.connector.config.server)) (not (empty .Values.connector.config.accessKey)) }}
{{- end }}
