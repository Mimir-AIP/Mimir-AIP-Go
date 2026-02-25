{{/*
Expand the name of the chart.
*/}}
{{- define "mimir-aip.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "mimir-aip.fullname" -}}
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
Common labels applied to all resources.
*/}}
{{- define "mimir-aip.labels" -}}
helm.sh/chart: {{ printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
app.kubernetes.io/name: {{ include "mimir-aip.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels — used in matchLabels and pod label selectors.
*/}}
{{- define "mimir-aip.selectorLabels" -}}
app.kubernetes.io/name: {{ include "mimir-aip.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Resolve the image tag: prefer .Values.image.tag, fall back to Chart.AppVersion.
*/}}
{{- define "mimir-aip.imageTag" -}}
{{- default .Chart.AppVersion .Values.image.tag }}
{{- end }}

{{/*
Resolve the worker namespace: use .Values.orchestrator.workerNamespace if set,
otherwise fall back to the release namespace.
*/}}
{{- define "mimir-aip.workerNamespace" -}}
{{- if .Values.orchestrator.workerNamespace }}
{{- .Values.orchestrator.workerNamespace }}
{{- else }}
{{- .Release.Namespace }}
{{- end }}
{{- end }}
