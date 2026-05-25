{{/*
Expand the name of the chart.
*/}}
{{- define "webhull.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "webhull.fullname" -}}
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
Create chart label.
*/}}
{{- define "webhull.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels.
*/}}
{{- define "webhull.labels" -}}
helm.sh/chart: {{ include "webhull.chart" . }}
{{ include "webhull.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Base selector labels (name + instance).
*/}}
{{- define "webhull.selectorLabels" -}}
app.kubernetes.io/name: {{ include "webhull.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Image tag — uses .Values.image.tag when set, otherwise .Chart.AppVersion.
*/}}
{{- define "webhull.imageTag" -}}
{{- .Values.image.tag | default .Chart.AppVersion }}
{{- end }}

{{/*
Gateway name — uses .Values.gateway.name when set, otherwise chart fullname.
*/}}
{{- define "webhull.gatewayName" -}}
{{- .Values.gateway.name | default (include "webhull.fullname" .) }}
{{- end }}

{{/*
TLS secret name — uses .Values.gateway.tlsSecretName when set, otherwise fullname-tls.
*/}}
{{- define "webhull.tlsSecretName" -}}
{{- .Values.gateway.tlsSecretName | default (printf "%s-tls" (include "webhull.fullname" .)) }}
{{- end }}
