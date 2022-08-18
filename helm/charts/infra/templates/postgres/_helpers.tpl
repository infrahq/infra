{{/*
Expand the name of the chart.
*/}}
{{- define "postgres.name" -}}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if hasSuffix .Values.postgres.componentName  $name }}
{{- $name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" $name .Values.postgres.componentName | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "postgres.fullname" -}}
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
{{- if hasSuffix .Values.postgres.componentName $name }}
{{- $name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" $name .Values.postgres.componentName | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "postgres.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "postgres.labels" -}}
helm.sh/chart: {{ include "postgres.chart" . }}
app.infrahq.com/component: {{ .Values.postgres.componentName }}
{{ include "postgres.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- if or .Values.postgres.labels .Values.global.labels }}
{{ merge .Values.postgres.labels .Values.global.postgres.labels | toYaml }}
{{- end }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "postgres.selectorLabels" -}}
app.kubernetes.io/name: {{ include "postgres.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Pod labels
*/}}
{{- define "postgres.podLabels" -}}
{{- include "postgres.selectorLabels" . }}
{{- if or .Values.postgres.podLabels .Values.global.podLabels }}
{{ merge .Values.postgres.podLabels .Values.global.podLabels | toYaml }}
{{- end }}
{{- end }}

{{/*
Pod annotations
*/}}
{{- define "postgres.podAnnotations" -}}
{{- if or .Values.postgres.podAnnotations .Values.global.podAnnotations }}
{{ merge .Values.postgres.podAnnotations .Values.global.podAnnotations | toYaml }}
{{- end }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "postgres.serviceAccountName" -}}
{{- if .Values.postgres.serviceAccount.create }}
{{- default (include "postgres.fullname" .) .Values.postgres.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.postgres.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
postgres 'env' values. Merges global and local values.
*/}}
{{- define "postgres.env" -}}
{{- $env := concat .Values.postgres.env .Values.global.env }}
{{- $env = append $env (dict "name" "POSTGRES_DB" "value" .Values.postgres.dbName) }}
{{- $env = append $env (dict "name" "POSTGRES_USER" "value" .Values.postgres.dbUsername) }}
{{- if .Values.postgres.dbPasswordSecret -}}
{{- $env = append $env (dict "name" "POSTGRES_PASSWORD" "valueFrom" (dict "secretKeyRef" (dict "name" .Values.postgres.dbPasswordSecret "key" "password"))) }}
{{- else }}
{{- $env = append $env (dict "name" "POSTGRES_PASSWORD" "valueFrom" (dict "secretKeyRef" (dict "name" (include "postgres.fullname" .) "key" "password"))) }}
{{- end }}
{{- $env | uniq | toYaml }}
{{- end }}

{{/*
postgres 'envFrom' values. Merges global and local values.
*/}}
{{- define "postgres.envFrom" -}}
{{- concat .Values.postgres.envFrom .Values.global.envFrom | uniq | toYaml }}
{{- end }}

{{/*
Infer whether postgres should be deployed based on postgres.enabled, server.enabled, and external postgres connection configurations.
*/}}
{{- define "postgres.enabled" -}}
{{- and .Values.postgres.enabled .Values.server.enabled (not .Values.server.config.dbHost) }}
{{- end }}
