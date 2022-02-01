{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "infra.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "infra.labels" -}}
helm.sh/chart: {{ include "infra.chart" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/component: {{ .Chart.Name }}
{{- if .Values.global.labels }}
{{ toYaml .Values.global.labels }}
{{- end }}
{{- end }}

{{/*
Create an server access key. Look for an existing secret and use its value. If the secret
does not exist, randomly generate a value.
*/}}
{{- define "infra.accessKey" -}}
{{- $secret := lookup "v1" "Secret" .Release.Namespace (printf "%s-access-key" .Release.Name) }}
{{- if $secret }}
{{- index $secret "data" "access-key" | b64dec }}
{{- else }}
{{- randAlphaNum 10 }}.{{ randAlphaNum 24 }}
{{- end }}
{{- end }}
