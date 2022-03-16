{{/*
Expand the name of the chart.
*/}}
{{- define "engine.name" -}}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if hasSuffix .Values.engine.componentName  $name }}
{{- $name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" $name .Values.engine.componentName | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "engine.fullname" -}}
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
{{- if hasSuffix .Values.engine.componentName $name }}
{{- $name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" $name .Values.engine.componentName | trunc 63 | trimSuffix "-" }}
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
{{- if or .Values.engine.labels .Values.global.labels }}
{{ merge .Values.engine.labels .Values.global.labels | toYaml }}
{{- end }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "engine.selectorLabels" -}}
app.kubernetes.io/name: {{ include "engine.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: {{ .Values.engine.componentName }}
{{- end }}

{{/*
Pod labels
*/}}
{{- define "engine.podLabels" -}}
{{- include "engine.selectorLabels" . }}
{{- if or .Values.engine.podLabels .Values.global.podLabels }}
{{ merge .Values.engine.podLabels .Values.global.podLabels | toYaml }}
{{- end }}
{{- end }}

{{/*
Pod annotations
*/}}
{{- define "engine.podAnnotations" -}}
rollme: {{ include (print .Template.BasePath "/engine/configmap.yaml") . | sha1sum }}
{{- if or .Values.engine.podAnnotations .Values.global.podAnnotations }}
{{ merge .Values.engine.podAnnotations .Values.global.podAnnotations | toYaml }}
{{- end }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "engine.serviceAccountName" -}}
{{- if .Values.engine.serviceAccount.create }}
{{- default (include "engine.fullname" .) .Values.engine.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.engine.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Engine image repository.
If global value is present, use global value. Otherwise, use local value.
*/}}
{{- define "engine.image.repository" -}}
{{- if .Values.global.image }}
{{- .Values.global.image.repository | default .Values.engine.image.repository }}
{{- else }}
{{- .Values.engine.image.repository }}
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
{{- .Values.engine.image.tag | default .Values.global.image.tag | default .Chart.AppVersion }}
{{- else }}
{{- .Values.engine.image.tag | default .Chart.AppVersion }}
{{- end }}
{{- end }}

{{/*
Engine image pull policy.
If global value is present, use global value. Otherwise, use local value.
*/}}
{{- define "engine.image.pullPolicy" -}}
{{- if .Values.global.image }}
{{- .Values.global.image.pullPolicy | default .Values.engine.image.pullPolicy }}
{{- else }}
{{- .Values.engine.image.pullPolicy }}
{{- end }}
{{- end }}

{{/*
Engine image pull secrets.
If global value is present, use global value. Otherwise, use local value.
*/}}
{{- define "engine.imagePullSecrets" -}}
{{ concat .Values.engine.imagePullSecrets .Values.global.imagePullSecrets | uniq | toYaml }}
{{- end }}

{{/*
Create an admin access key. If one is defined through values, use it. Otherwise look for an
existing secret and use its password. If the secret does not exist, randomly generate a password.
*/}}
{{- define "engine.accessKey" -}}
{{- if .Values.engine.config.accessKey }}
{{- .Values.engine.config.accessKey }}
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
Infer whether Infra engine should be deployed based on engine.enabled, engine.config.server, and engine.config.accessKey.
*/}}
{{- define "engine.enabled" -}}
{{- or .Values.engine.enabled (not (empty .Values.engine.config.server)) (not (empty .Values.engine.config.accessKey)) }}
{{- end }}
