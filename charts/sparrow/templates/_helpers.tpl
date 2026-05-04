{{/*
Chart name, truncated to 63 chars (K8s label limit).
*/}}
{{- define "sparrow.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Fully qualified app name: <release>-<chart>, truncated to 63 chars.
If release name already contains chart name, don't duplicate it.
*/}}
{{- define "sparrow.fullname" -}}
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
Chart label value: "<name>-<version>"
*/}}
{{- define "sparrow.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels applied to every resource.
*/}}
{{- define "sparrow.labels" -}}
helm.sh/chart: {{ include "sparrow.chart" . }}
{{ include "sparrow.selectorLabels" . }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels — used in Deployment.spec.selector and Service.spec.selector.
Must not change after initial install.
*/}}
{{- define "sparrow.selectorLabels" -}}
app.kubernetes.io/name: {{ include "sparrow.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
ServiceAccount name.
*/}}
{{- define "sparrow.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "sparrow.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Secret name — either the user's existing secret or the chart-created one.
*/}}
{{- define "sparrow.secretName" -}}
{{- if .Values.secrets.existingSecret }}
{{- .Values.secrets.existingSecret }}
{{- else }}
{{- include "sparrow.fullname" . }}
{{- end }}
{{- end }}

{{/*
PostgreSQL service hostname (internal cluster DNS).
*/}}
{{- define "sparrow.postgresql.host" -}}
{{- printf "%s-postgresql" (include "sparrow.fullname" .) }}
{{- end }}

{{/*
Auto-constructed DATABASE_URL when using built-in PostgreSQL.
sslmode=disable is safe for in-cluster traffic.
*/}}
{{- define "sparrow.postgresql.url" -}}
{{- printf "postgres://%s:%s@%s:5432/%s?sslmode=%s" .Values.postgresql.auth.username .Values.postgresql.auth.password (include "sparrow.postgresql.host" .) .Values.postgresql.auth.database (default "disable" .Values.postgresql.auth.sslmode) }}
{{- end }}
