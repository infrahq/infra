{{/*
Expand the name of the chart.
*/}}
{{- define "ui.name" -}}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if hasSuffix .Values.ui.componentName  $name }}
{{- $name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" $name .Values.ui.componentName | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "ui.fullname" -}}
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
{{- if hasSuffix .Values.ui.componentName $name }}
{{- $name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" $name .Values.ui.componentName | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "ui.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "ui.labels" -}}
helm.sh/chart: {{ include "ui.chart" . }}
app.infrahq.com/component: {{ .Values.ui.componentName }}
{{ include "ui.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- if or .Values.ui.labels .Values.global.labels }}
{{ merge .Values.ui.labels .Values.global.ui.labels | toYaml }}
{{- end }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "ui.selectorLabels" -}}
app.kubernetes.io/name: {{ include "ui.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Pod labels
*/}}
{{- define "ui.podLabels" -}}
{{- include "ui.selectorLabels" . }}
{{- if or .Values.ui.podLabels .Values.global.podLabels }}
{{ merge .Values.ui.podLabels .Values.global.podLabels | toYaml }}
{{- end }}
{{- end }}

{{/*
Pod annotations
*/}}
{{- define "ui.podAnnotations" -}}
{{- if or .Values.ui.podAnnotations .Values.global.podAnnotations }}
{{ merge .Values.ui.podAnnotations .Values.global.podAnnotations | toYaml }}
{{- end }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "ui.serviceAccountName" -}}
{{- if .Values.ui.serviceAccount.create }}
{{- default (include "ui.fullname" .) .Values.ui.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.ui.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
UI image repository.
If global value is present, use global value. Otherwise, use local value.
*/}}
{{- define "ui.image.repository" -}}
{{- if .Values.global.image }}
{{- .Values.global.image.repository | default .Values.ui.image.repository }}
{{- else }}
{{- .Values.ui.image.repository }}
{{- end }}
{{- end }}

{{/*
UI image tag.
If a local override exists, use the local override. Otherwise, if a global
override exists, use the global override.  If `image.tag` does not exist,
use AppVersion defined in Chart.
*/}}
{{- define "ui.image.tag" -}}
{{- if .Values.global.image }}
{{- .Values.ui.image.tag | default .Values.global.image.tag | default .Chart.AppVersion }}
{{- else }}
{{- .Values.ui.image.tag | default .Chart.AppVersion }}
{{- end }}
{{- end }}

{{/*
UI image pull policy.
If global value is present, use global value. Otherwise, use local value.
*/}}
{{- define "ui.image.pullPolicy" -}}
{{- if .Values.global.image }}
{{- .Values.global.image.pullPolicy | default .Values.ui.image.pullPolicy }}
{{- else }}
{{- .Values.ui.image.pullPolicy }}
{{- end }}
{{- end }}

{{/*
UI image pull secrets. Merges global and local values.
*/}}
{{- define "ui.imagePullSecrets" -}}
{{ concat .Values.ui.imagePullSecrets .Values.global.imagePullSecrets | uniq | toYaml }}
{{- end }}

{{/*
UI 'env' values. Merges global and local values.
*/}}
{{- define "ui.env" -}}
{{- $env := concat .Values.ui.env .Values.global.env }}

{{- concat $env | uniq | toYaml }}
{{- end }}

{{/*
UI 'envFrom' values. Merges global and local values.
*/}}
{{- define "ui.envFrom" -}}
{{- concat .Values.ui.envFrom .Values.global.envFrom | uniq | toYaml }}
{{- end }}

{{/*
Infer whether ui should be deployed based on ui.enabled and connector.config.ui.
*/}}
{{- define "ui.enabled" -}}
{{- and .Values.server.enabled (not .Values.connector.config.server) }}
{{- end }}
