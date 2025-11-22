{{/*
Expand the name of the chart.
*/}}
{{- define "unity-collab.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "unity-collab.fullname" -}}
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
Create chart name and version as used by the chart label.
*/}}
{{- define "unity-collab.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "unity-collab.labels" -}}
helm.sh/chart: {{ include "unity-collab.chart" . }}
{{ include "unity-collab.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
environment: {{ .Values.global.environment }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "unity-collab.selectorLabels" -}}
app.kubernetes.io/name: {{ include "unity-collab.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Auth Service labels
*/}}
{{- define "unity-collab.authService.labels" -}}
{{ include "unity-collab.labels" . }}
app.kubernetes.io/component: auth-service
{{- end }}

{{/*
Session Service labels
*/}}
{{- define "unity-collab.sessionService.labels" -}}
{{ include "unity-collab.labels" . }}
app.kubernetes.io/component: session-service
{{- end }}

{{/*
Asset Service labels
*/}}
{{- define "unity-collab.assetService.labels" -}}
{{ include "unity-collab.labels" . }}
app.kubernetes.io/component: asset-service
{{- end }}

{{/*
Sync Service labels
*/}}
{{- define "unity-collab.syncService.labels" -}}
{{ include "unity-collab.labels" . }}
app.kubernetes.io/component: sync-service
{{- end }}

{{/*
Gateway labels
*/}}
{{- define "unity-collab.gateway.labels" -}}
{{ include "unity-collab.labels" . }}
app.kubernetes.io/component: gateway
{{- end }}

{{/*
PostgreSQL labels
*/}}
{{- define "unity-collab.postgresql.labels" -}}
{{ include "unity-collab.labels" . }}
app.kubernetes.io/component: postgresql
{{- end }}

{{/*
Redis labels
*/}}
{{- define "unity-collab.redis.labels" -}}
{{ include "unity-collab.labels" . }}
app.kubernetes.io/component: redis
{{- end }}

{{/*
Image pull secrets
*/}}
{{- define "unity-collab.imagePullSecrets" -}}
{{- if .Values.global.imagePullSecrets }}
imagePullSecrets:
{{- range .Values.global.imagePullSecrets }}
  - name: {{ . }}
{{- end }}
{{- end }}
{{- end }}
