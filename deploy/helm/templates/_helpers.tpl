{{/*
Expand the name of the chart.
*/}}
{{- define "webcore.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "webcore.fullname" -}}
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
{{- define "webcore.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels.
*/}}
{{- define "webcore.labels" -}}
helm.sh/chart: {{ include "webcore.chart" . }}
{{ include "webcore.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Base selector labels (name + instance).
*/}}
{{- define "webcore.selectorLabels" -}}
app.kubernetes.io/name: {{ include "webcore.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Image tag — uses .Values.image.tag when set, otherwise .Chart.AppVersion.
*/}}
{{- define "webcore.imageTag" -}}
{{- .Values.image.tag | default .Chart.AppVersion }}
{{- end }}

{{/*
Gateway name — uses .Values.gateway.name when set, otherwise chart fullname.
*/}}
{{- define "webcore.gatewayName" -}}
{{- .Values.gateway.name | default (include "webcore.fullname" .) }}
{{- end }}

{{/*
TLS secret name — uses .Values.gateway.tlsSecretName when set, otherwise fullname-tls.
*/}}
{{- define "webcore.tlsSecretName" -}}
{{- .Values.gateway.tlsSecretName | default (printf "%s-tls" (include "webcore.fullname" .)) }}
{{- end }}
