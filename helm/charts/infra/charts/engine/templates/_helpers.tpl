{{/*
Expand the name of the chart.
*/}}
{{- define "engine.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "engine.fullname" -}}
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
{{- define "engine.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "engine.labels" -}}
helm.sh/chart: {{ include "engine.chart" . }}
{{ include "engine.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/component: {{ .Chart.Name }}
{{- if or .Values.labels .Values.global.labels }}
{{ .Values.global.labels | default dict | merge .Values.labels | toYaml }}
{{- end }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "engine.selectorLabels" -}}
app.kubernetes.io/name: {{ include "engine.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Pod labels
*/}}
{{- define "engine.podLabels" -}}
{{- include "engine.selectorLabels" . }}
{{- if or .Values.podLabels .Values.global.podLabels }}
{{ .Values.global.podLabels | default dict | merge .Values.podLabels | toYaml }}
{{- end }}
{{- end }}

{{/*
Pod annotations
*/}}
{{- define "engine.podAnnotations" -}}
rollme: {{ include (print .Template.BasePath "/configmap.yaml") . | sha1sum }}
{{- if or .Values.podAnnotations .Values.global.podAnnotations }}
{{- .Values.global.podAnnotations | default dict | merge .Values.podAnnotations | toYaml }}
{{- end }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "engine.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "engine.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Engine image repository.
If global value is present, use global value. Otherwise, use local value.
*/}}
{{- define "engine.image.repository" -}}
{{- if .Values.global.image }}
{{- .Values.global.image.repository | default .Values.image.repository }}
{{- else }}
{{- .Values.image.repository }}
{{- end }}
{{- end }}

{{/*
Engine image tag.
If a local override exists, use the local override. Otherwise, if a global
override exists, use the global override.  If `image.tag` does not exist,
use AppVersion defined in Chart.
*/}}
{{- define "engine.image.tag" -}}
{{- if .Values.global.image }}
{{- .Values.image.tag | default .Values.global.image.tag | default .Chart.AppVersion }}
{{- else }}
{{- .Values.image.tag | default .Chart.AppVersion }}
{{- end }}
{{- end }}

{{/*
Engine image pull policy.
If global value is present, use global value. Otherwise, use local value.
*/}}
{{- define "engine.image.pullPolicy" -}}
{{- if .Values.global.image }}
{{- .Values.global.image.pullPolicy | default .Values.image.pullPolicy }}
{{- else }}
{{- .Values.image.pullPolicy }}
{{- end }}
{{- end }}

{{/*
Engine image pull secrets.
If global value is present, use global value. Otherwise, use local value.
*/}}
{{- define "engine.imagePullSecrets" -}}
{{- .Values.global.imagePullSecrets | default list | concat .Values.imagePullSecrets | uniq | toYaml }}
{{- end }}
