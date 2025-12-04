{{/*
Expand the name of the chart.
*/}}
{{- define "observer.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "observer.fullname" -}}
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
{{- define "observer.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "observer.labels" -}}
helm.sh/chart: {{ include "observer.chart" . }}
{{ include "observer.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "observer.selectorLabels" -}}
app.kubernetes.io/name: {{ include "observer.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "observer.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "observer.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Get the image registry
*/}}
{{- define "observer.image.registry" -}}
{{- .Values.image.registry }}
{{- end }}

{{/*
Get the image repository
*/}}
{{- define "observer.image.repository" -}}
{{- .Values.image.repository }}
{{- end }}

{{/*
Get the image tag
*/}}
{{- define "observer.image.tag" -}}
{{- .Values.image.tag | default .Chart.AppVersion }}
{{- end }}

{{/*
Get the image pull policy - use Always for mutable tags, IfNotPresent for immutable
*/}}
{{- define "observer.image.pullPolicy" -}}
{{- $tag := include "observer.image.tag" . -}}
{{- if .Values.image.pullPolicy -}}
{{- .Values.image.pullPolicy }}
{{- else if or (hasSuffix "latest" $tag) (hasSuffix "main" $tag) (hasSuffix "develop" $tag) -}}
Always
{{- else -}}
IfNotPresent
{{- end -}}
{{- end }}

{{/*
Get the full image name for a component
*/}}
{{- define "observer.image" -}}
{{- $registry := include "observer.image.registry" . }}
{{- $repository := include "observer.image.repository" . }}
{{- $tag := include "observer.image.tag" . }}
{{- $component := .component }}
{{- printf "%s/%s/%s:%s" $registry $repository $component $tag }}
{{- end }}

{{/*
Database connection string
*/}}
{{- define "observer.database.url" -}}
{{- if .Values.mongodb.enabled }}
{{- printf "mongodb://%s:%s@%s-mongodb:27017/%s?authSource=admin" .Values.mongodb.auth.rootUser .Values.mongodb.auth.rootPassword (include "observer.fullname" .) .Values.mongodb.auth.database }}
{{- else }}
{{- printf "mongodb://%s:%s@%s:%d/%s?authSource=admin" .Values.externalDatabase.username .Values.externalDatabase.password .Values.externalDatabase.host (int .Values.externalDatabase.port) .Values.externalDatabase.database }}
{{- end }}
{{- end }}

{{/*
NATS connection URL
*/}}
{{- define "observer.nats.url" -}}
{{- if .Values.nats.enabled }}
{{- printf "nats://%s-nats:4222" (include "observer.fullname" .) }}
{{- else }}
{{- .Values.externalNats.url }}
{{- end }}
{{- end }}
