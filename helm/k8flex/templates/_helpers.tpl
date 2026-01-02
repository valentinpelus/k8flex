{{/*
Expand the name of the chart.
*/}}
{{- define "k8flex.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "k8flex.fullname" -}}
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
{{- define "k8flex.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "k8flex.labels" -}}
helm.sh/chart: {{ include "k8flex.chart" . }}
{{ include "k8flex.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "k8flex.selectorLabels" -}}
app.kubernetes.io/name: {{ include "k8flex.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "k8flex.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "k8flex.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Generate webhook auth token or use existing one
This will generate a random token on first install and persist it on upgrades
*/}}
{{- define "k8flex.webhookAuthToken" -}}
{{- if .Values.webhook.authToken -}}
{{- .Values.webhook.authToken -}}
{{- else -}}
{{- $secret := (lookup "v1" "Secret" .Release.Namespace (printf "%s-secrets" (include "k8flex.fullname" .))) -}}
{{- if and $secret $secret.data -}}
{{- if hasKey $secret.data "WEBHOOK_AUTH_TOKEN" -}}
{{- index $secret.data "WEBHOOK_AUTH_TOKEN" | b64dec -}}
{{- else -}}
{{- randAlphaNum 64 -}}
{{- end -}}
{{- else -}}
{{- randAlphaNum 64 -}}
{{- end -}}
{{- end -}}
{{- end }}
