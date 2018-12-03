{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
*/}}
{{- define "fullname" -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Expand the name of the chart.
*/}}
{{- define "clustername" -}}
{{- $name := default .Release.Name .Values.nameOverride -}}
{{- printf "%s" $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Expand the service name.
*/}}
{{- define "service-name" -}}
{{- $cname := default .Chart.Name .Values.nameOverride -}}
{{- $fullname := printf "%s-%s" .Release.Name $cname | trunc 63 | trimSuffix "-" -}}
{{- $name := default $fullname .Values.serviceName -}}
{{- printf "%s" $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Expand the service account name.
*/}}
{{- define "serviceaccount" -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- $clustername := printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- $account := default $clustername .Values.serviceAccount -}}
{{- printf "%s" $account | trunc 63 | trimSuffix "-" -}}
{{- end -}}