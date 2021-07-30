{{- define "infra.defaultApiKey" -}}
{{- $secret := (lookup "v1" "Secret" .Release.Namespace "infra-engine" ) -}}
  {{- if $secret -}}
    {{-  index $secret "data" "api-key" | b64dec -}}
  {{- else -}}
    {{- (randAlphaNum 24) -}}
  {{- end -}}
{{- end -}}
