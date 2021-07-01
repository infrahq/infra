{{- define "infra.defaultApiKey" -}}
{{- $secret := (lookup "v1" "Secret" .Release.Namespace "infra" ) -}}
  {{- if $secret -}}
    {{-  index $secret "data" "default-api-key" | b64dec -}}
  {{- else -}}
    {{- (randAlphaNum 24) -}}
  {{- end -}}
{{- end -}}
