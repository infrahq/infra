{{/*
Expand the name of the chart.
*/}}
{{- define "server.name" -}}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if hasSuffix .Values.server.componentName  $name }}
{{- $name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" $name .Values.server.componentName | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "server.fullname" -}}
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
{{- if hasSuffix .Values.server.componentName $name }}
{{- $name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" $name .Values.server.componentName | trunc 63 | trimSuffix "-" }}
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
{{- if or .Values.server.labels .Values.global.labels }}
{{ merge .Values.server.labels .Values.global.server.labels | toYaml }}
{{- end }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "server.selectorLabels" -}}
app.kubernetes.io/name: {{ include "server.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: {{ .Values.server.componentName }}
{{- end }}

{{/*
Pod labels
*/}}
{{- define "server.podLabels" -}}
{{- include "server.selectorLabels" . }}
{{- if or .Values.server.podLabels .Values.global.podLabels }}
{{ merge .Values.server.podLabels .Values.global.podLabels | toYaml }}
{{- end }}
{{- end }}

{{/*
Pod annotations
*/}}
{{- define "server.podAnnotations" -}}
rollme: {{ include (print .Template.BasePath "/server/configmap.yaml") . | sha1sum }}
{{- if or .Values.server.podAnnotations .Values.global.podAnnotations }}
{{ merge .Values.server.podAnnotations .Values.global.podAnnotations | toYaml }}
{{- end }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "server.serviceAccountName" -}}
{{- if .Values.server.serviceAccount.create }}
{{- default (include "server.fullname" .) .Values.server.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.server.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Infra server image repository.
If global value is present, use global value. Otherwise, use local value.
*/}}
{{- define "server.image.repository" -}}
{{- if .Values.global.image }}
{{- .Values.global.image.repository | default .Values.server.image.repository }}
{{- else }}
{{- .Values.server.image.repository }}
{{- end }}
{{- end }}

{{/*
Infra server image tag.
If a local override exists, use the local override. Otherwise, if a global
override exists, use the global override.  If `image.tag` does not exist,
use AppVersion defined in Chart.
*/}}
{{- define "server.image.tag" -}}
{{- if .Values.global.image }}
{{- .Values.server.image.tag | default .Values.global.image.tag | default .Chart.AppVersion }}
{{- else }}
{{- .Values.server.image.tag | default .Chart.AppVersion }}
{{- end }}
{{- end }}

{{/*
Infra server image pull policy.
If global value is present, use global value. Otherwise, use local value.
*/}}
{{- define "server.image.pullPolicy" -}}
{{- if .Values.global.image }}
{{- .Values.global.image.pullPolicy | default .Values.server.image.pullPolicy }}
{{- else }}
{{- .Values.server.image.pullPolicy }}
{{- end }}
{{- end }}

{{/*
Infra server image pull secrets.
If global value is present, use global value. Otherwise, use local value.
*/}}
{{- define "server.imagePullSecrets" -}}
{{ concat .Values.server.imagePullSecrets .Values.global.imagePullSecrets | uniq | toYaml }}
{{- end }}

{{/*
Create an admin access key. If one is defined through values, use it. Otherwise look for an
existing secret and use its password. If the secret does not exist, randomly generate a password.
*/}}
{{- define "server.adminAccessKey" -}}
{{- if .Values.server.config.adminAccessKey }}
{{- .Values.server.config.adminAccessKey }}
{{- else }}
{{- $secret := lookup "v1" "Secret" .Release.Namespace (printf "%s-admin-access-key" .Release.Name) }}
{{- if $secret }}
{{- index $secret "data" "access-key" | b64dec }}
{{- else }}
{{- randAlphaNum 10 }}.{{ randAlphaNum 24 }}
{{- end }}
{{- end }}
{{- end }}
