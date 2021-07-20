{{/*
Return the target Kubernetes version
*/}}
{{- define "infra.kubeVersion" -}}
  {{- default .Capabilities.KubeVersion.Version .Values.kubeVersionOverride }}
{{- end -}}

{{/*
Return the appropriate apiVersion for ingress
*/}}
{{- define "infra.ingress.apiVersion" -}}
{{- if semverCompare "<1.19-0" (include "infra.kubeVersion" $) -}}
{{- print "networking.k8s.io/v1beta1" -}}
{{- else -}}
{{- print "networking.k8s.io/v1" -}}
{{- end -}}
{{- end -}}
