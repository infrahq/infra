{{/*
Expand the name of the chart.
*/}}
{{- define "server.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "server.fullname" -}}
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
{{- define "server.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "server.labels" -}}
helm.sh/chart: {{ include "server.chart" . }}
{{ include "server.selectorLabels" . }}
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
{{- define "server.selectorLabels" -}}
app.kubernetes.io/name: {{ include "server.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Pod labels
*/}}
{{- define "server.podLabels" -}}
{{- include "server.selectorLabels" . }}
{{- if or .Values.podLabels .Values.global.podLabels }}
{{ .Values.global.podLabels | default dict | merge .Values.podLabels | toYaml }}
{{- end }}
{{- end }}

{{/*
Pod annotations
*/}}
{{- define "server.podAnnotations" -}}
rollme: {{ include (print .Template.BasePath "/configmap.yaml") . | sha1sum }}
{{- if or .Values.podAnnotations .Values.global.podAnnotations }}
{{- .Values.global.podAnnotations | default dict | merge .Values.podAnnotations | toYaml }}
{{- end }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "server.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "server.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Server image repository.
If global value is present, use global value. Otherwise, use local value.
*/}}
{{- define "server.image.repository" -}}
{{- if .Values.global.image }}
{{- .Values.global.image.repository | default .Values.image.repository }}
{{- else }}
{{- .Values.image.repository }}
{{- end }}
{{- end }}

{{/*
Server image tag.
If a local override exists, use the local override. Otherwise, if a global
override exists, use the global override.  If `image.tag` does not exist,
use AppVersion defined in Chart.
*/}}
{{- define "server.image.tag" -}}
{{- if .Values.global.image }}
{{- .Values.image.tag | default .Values.global.image.tag | default .Chart.AppVersion }}
{{- else }}
{{- .Values.image.tag | default .Chart.AppVersion }}
{{- end }}
{{- end }}

{{/*
Server image pull policy.
If global value is present, use global value. Otherwise, use local value.
*/}}
{{- define "server.image.pullPolicy" -}}
{{- if .Values.global.image }}
{{- .Values.global.image.pullPolicy | default .Values.image.pullPolicy }}
{{- else }}
{{- .Values.image.pullPolicy }}
{{- end }}
{{- end }}

{{/*
Server image pull secrets.
If global value is present, use global value. Otherwise, use local value.
*/}}
{{- define "server.imagePullSecrets" -}}
{{- .Values.global.imagePullSecrets | default list | concat .Values.imagePullSecrets | uniq | toYaml }}
{{- end }}

{{/*
Create an system access key. If one is defined through values, use it. Otherwise look for an
existing secret and use its password. If the secret does not exist, randomly generate a password.
*/}}
{{- define "server.systemAccessKey" -}}
{{- if .Values.systemAccessKey }}
{{- .Values.systemAccessKey }}
{{- else }}
{{- $secret := lookup "v1" "Secret" .Release.Namespace (printf "%s-system-access-key" .Release.Name) }}
{{- if $secret }}
{{- index $secret "data" "access-key" | b64dec }}
{{- else }}
{{- randAlphaNum 10 }}.{{ randAlphaNum 24 }}
{{- end }}
{{- end }}
{{- end }}
