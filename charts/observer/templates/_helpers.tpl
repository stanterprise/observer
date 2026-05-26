{{/*
Validate deployment mode early so operators see a clear error instead of a broken render.
*/}}
{{- define "observer.validateMode" -}}
{{- if not (has .Values.mode (list "aio" "distributed")) -}}
{{- fail (printf "values.mode must be 'aio' or 'distributed'; got %q" .Values.mode) -}}
{{- end -}}
{{- end }}

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
Runtime Secret name for distributed dependency connection settings
*/}}
{{- define "observer.runtimeSecretName" -}}
{{- if .Values.runtime.existingSecret -}}
{{- .Values.runtime.existingSecret -}}
{{- else -}}
{{- printf "%s-runtime-env" (include "observer.fullname" .) -}}
{{- end -}}
{{- end }}

{{/*
Distributed workloads must not override chart-managed connection env vars.
*/}}
{{- define "observer.validateNoManagedConnectionEnv" -}}
{{- $path := .path -}}
{{- $env := .env | default dict -}}
{{- range $key := list "NATS_URL" "POSTGRES_DSN" "MONGODB_URI" -}}
{{- if hasKey $env $key -}}
{{- fail (printf "%s.%s is not supported in distributed mode; use runtime.existingSecret or canonical dependency values (externalNats.url, postgres.*, externalDatabase.*, or embedded dependency settings)" $path $key) -}}
{{- end -}}
{{- end -}}
{{- end }}

{{/*
Validate distributed-mode configuration before rendering resources.
*/}}
{{- define "observer.validateDistributedConfig" -}}
{{- if and (eq .Values.mode "distributed") .Values.distributed.enabled -}}
{{- include "observer.validateNoManagedConnectionEnv" (dict "path" "distributed.ingestion.env" "env" (.Values.distributed.ingestion.env | default dict)) -}}
{{- include "observer.validateNoManagedConnectionEnv" (dict "path" "distributed.api.env" "env" (.Values.distributed.api.env | default dict)) -}}
{{- include "observer.validateNoManagedConnectionEnv" (dict "path" "distributed.processor.env" "env" (.Values.distributed.processor.env | default dict)) -}}
{{- end -}}
{{- end }}

{{/*
Render workload env entries while skipping chart-managed keys.
*/}}
{{- define "observer.renderFilteredEnv" -}}
{{- $env := .env | default dict -}}
{{- $excluded := .excluded | default list -}}
{{- range $key, $value := $env }}
{{- if not (has $key $excluded) }}
- name: {{ $key }}
  value: {{ $value | quote }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Distributed runtime NATS URL.
Source-of-truth comes from canonical chart values only.
*/}}
{{- define "observer.runtime.natsUrl" -}}
{{- include "observer.nats.url" . -}}
{{- end }}

{{/*
Distributed runtime PostgreSQL DSN.
Source-of-truth comes from canonical chart values only.
*/}}
{{- define "observer.runtime.postgresDsn" -}}
{{- include "observer.postgres.dsn" . -}}
{{- end }}

{{/*
Distributed runtime MongoDB URI.
Source-of-truth comes from canonical chart values only.
*/}}
{{- define "observer.runtime.mongodbUri" -}}
{{- include "observer.database.url" . -}}
{{- end }}

{{/*
Database connection string (MongoDB URI)
*/}}
{{- define "observer.database.url" -}}
{{- if .Values.mongodb.enabled }}
{{- $user := index .Values.mongodb.auth.usernames 0 | default "observer" }}
{{- $password := index .Values.mongodb.auth.passwords 0 | default "password" }}
{{- $database := index .Values.mongodb.auth.databases 0 | default "observer" }}
{{- printf "mongodb://%s:%s@%s-mongodb:27017/%s?authSource=%s" $user $password (include "observer.fullname" .) $database $database }}
{{- else }}
{{- $host := required "externalDatabase.host is required when mongodb.enabled=false" .Values.externalDatabase.host }}
{{- $authSource := .Values.externalDatabase.authSource | default "admin" }}
{{- printf "mongodb://%s:%s@%s:%d/%s?authSource=%s" .Values.externalDatabase.username .Values.externalDatabase.password $host (int .Values.externalDatabase.port) .Values.externalDatabase.database $authSource }}
{{- end }}
{{- end }}

{{/*
NATS connection URL
*/}}
{{- define "observer.nats.url" -}}
{{- if .Values.nats.enabled }}
{{- printf "nats://%s-nats:4222" (include "observer.fullname" .) }}
{{- else }}
{{- required "externalNats.url is required when nats.enabled=false" .Values.externalNats.url }}
{{- end }}
{{- end }}

{{/*
PostgreSQL Host
*/}}
{{- define "observer.postgres.host" -}}
{{- if .Values.postgresql.enabled -}}
{{- printf "%s-postgresql" (include "observer.fullname" .) -}}
{{- else -}}
{{- required "postgres.host is required when postgresql.enabled=false" .Values.postgres.host -}}
{{- end -}}
{{- end }}

{{/*
PostgreSQL Port
*/}}
{{- define "observer.postgres.port" -}}
{{- if .Values.postgresql.enabled -}}
{{- .Values.postgresql.primary.service.ports.postgresql | default 5432 -}}
{{- else -}}
{{- .Values.postgres.port | default 5432 -}}
{{- end -}}
{{- end }}

{{/*
PostgreSQL User
*/}}
{{- define "observer.postgres.user" -}}
{{- if .Values.postgresql.enabled -}}
{{- .Values.postgresql.auth.username | default "observer" -}}
{{- else -}}
{{- .Values.postgres.username | default "observer" -}}
{{- end -}}
{{- end }}

{{/*
PostgreSQL Password
*/}}
{{- define "observer.postgres.password" -}}
{{- if .Values.postgresql.enabled -}}
{{- .Values.postgresql.auth.password | default "password" -}}
{{- else -}}
{{- .Values.postgres.password | default "password" -}}
{{- end -}}
{{- end }}

{{/*
PostgreSQL Database
*/}}
{{- define "observer.postgres.db" -}}
{{- if .Values.postgresql.enabled -}}
{{- .Values.postgresql.auth.database | default "observer" -}}
{{- else -}}
{{- .Values.postgres.database | default "observer" -}}
{{- end -}}
{{- end }}

{{/*
PostgreSQL DSN
*/}}
{{- define "observer.postgres.dsn" -}}
{{- $host := include "observer.postgres.host" . -}}
{{- $port := include "observer.postgres.port" . -}}
{{- $username := include "observer.postgres.user" . -}}
{{- $password := include "observer.postgres.password" . -}}
{{- $database := include "observer.postgres.db" . -}}
{{- $sslmode := .Values.postgres.sslmode | default "disable" -}}
{{- if .Values.postgresql.enabled -}}
{{- $sslmode = "disable" -}}
{{- end -}}
{{- printf "postgres://%s:%s@%s:%v/%s?sslmode=%s" ($username | urlquery) ($password | urlquery) $host $port ($database | urlquery) $sslmode -}}
{{- end }}
