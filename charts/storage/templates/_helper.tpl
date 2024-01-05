{{- define "mainnode.filerUrl" -}}
{{- printf "drycc-storage-metanode-weed.%s.svc.%s:8888" .Release.Namespace .Values.global.clusterDomain }}
{{- end -}}

{{- define "mainnode.weedUrls" -}}
{{- $replicaCount := int .Values.mainnode.weed.replicas }}
{{- $clusterDomain := .Values.global.clusterDomain }}
{{- $messages := list -}}
{{ range $i := until $replicaCount }}
  {{- $messages = printf "drycc-storage-mainnode-weed-%d.drycc-storage-mainnode-weed.%s.svc.%s:9333" $i $.Release.Namespace $clusterDomain | append $messages -}}
{{ end }}
{{- $message := join "," $messages -}}
{{- printf "%s" $message }}
{{- end -}}

{{- define "mainnode.tipdUrls" -}}
{{- $replicaCount := int .Values.mainnode.tipd.replicas }}
{{- $clusterDomain := .Values.global.clusterDomain }}
{{- $messages := list -}}
{{ range $i := until $replicaCount }}
  {{- $messages = printf "http://drycc-storage-mainnode-tipd-%d.drycc-storage-mainnode-tipd.%s.svc.%s:2379" $i $.Release.Namespace $clusterDomain | append $messages -}}
{{ end }}
{{- $message := join "," $messages -}}
{{- printf "%s" $message }}
{{- end -}}

{{- /* keep randAlphaNum values consistent */ -}}
{{- define "storage.accesskey" -}}
  {{- if not (index .Release "secrets") -}}
    {{- $_ := set .Release "secrets" dict -}}
  {{- end -}}
  {{- if not (index .Release.secrets "accesskey") -}}
    {{- if .Values.accesskey | default "" | ne "" -}}
      {{- $_ := set .Release.secrets "accesskey" .Values.accesskey -}}
    {{- else -}}
      {{- $_ := set .Release.secrets "accesskey" (randAlphaNum 32) -}}
    {{- end -}}
  {{- end -}}
  {{- index .Release.secrets "accesskey" -}}
{{- end -}}

{{- /* keep randAlphaNum values consistent */ -}}
{{- define "storage.secretkey" -}}
  {{- if not (index .Release "secrets") -}}
    {{- $_ := set .Release "secrets" dict -}}
  {{- end -}}
  {{- if not (index .Release.secrets "secretkey") -}}
    {{- if .Values.secretkey | default "" | ne "" -}}
      {{- $_ := set .Release.secrets "secretkey" .Values.secretkey -}}
    {{- else -}}
      {{- $_ := set .Release.secrets "secretkey" (randAlphaNum 32) -}}
    {{- end -}}
  {{- end -}}
  {{- index .Release.secrets "secretkey" -}}
{{- end -}}
