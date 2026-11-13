{{/* vim: set filetype=mustache: */}}

{{- define "system_default_registry" -}}
{{- if .Values.global.cattle.systemDefaultRegistry -}}
{{- printf "%s/" .Values.global.cattle.systemDefaultRegistry -}}
{{- else -}}
{{- "" -}}
{{- end -}}
{{- end -}}

{{/*
Renders imagePullSecrets, accepting either object references ({ name: <secret> })
or plain strings
*/}}
{{- define "system-upgrade-controller.imagePullSecrets" -}}
{{- $pullSecrets := list -}}
{{- range .Values.global.cattle.imagePullSecrets -}}
  {{- if kindIs "map" . -}}
    {{- if .name -}}
      {{- $pullSecrets = append $pullSecrets .name -}}
    {{- end -}}
  {{- else if not (empty .) -}}
    {{- $pullSecrets = append $pullSecrets . -}}
  {{- end -}}
{{- end -}}
{{- if not (empty $pullSecrets) -}}
imagePullSecrets:
  {{- range $pullSecrets | uniq }}
  - name: {{ . }}
  {{- end }}
{{- end -}}
{{- end -}}
