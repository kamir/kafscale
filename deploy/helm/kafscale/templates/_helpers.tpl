{{- define "kafscale.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "kafscale.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := include "kafscale.name" . -}}
{{- if eq .Release.Name $name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{- define "kafscale.labels" -}}
app.kubernetes.io/name: {{ include "kafscale.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/part-of: kafscale
helm.sh/chart: {{ .Chart.Name }}-{{ .Chart.Version | replace "+" "_" }}
{{- end -}}

{{- define "kafscale.selectorLabels" -}}
app.kubernetes.io/name: {{ include "kafscale.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{- define "kafscale.operatorServiceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
{{- default (include "kafscale.fullname" .) .Values.serviceAccount.name -}}
{{- else -}}
{{- default "default" .Values.serviceAccount.name -}}
{{- end -}}
{{- end -}}

{{- define "kafscale.mcpServiceAccountName" -}}
{{- if .Values.mcp.serviceAccount.create -}}
{{- default (printf "%s-mcp" (include "kafscale.fullname" .)) .Values.mcp.serviceAccount.name -}}
{{- else -}}
{{- default "default" .Values.mcp.serviceAccount.name -}}
{{- end -}}
{{- end -}}

{{- define "kafscale.componentName" -}}
{{- printf "%s-%s" (include "kafscale.fullname" .root) .component | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "kafscale.componentSelectorLabels" -}}
app.kubernetes.io/name: {{ printf "%s-%s" (include "kafscale.name" .root) .component | trunc 63 | trimSuffix "-" }}
app.kubernetes.io/instance: {{ .root.Release.Name }}
app.kubernetes.io/component: {{ .component }}
{{- end -}}

{{/*
Permissions the operator needs in every namespace it reconciles. Rendered into
the ClusterRole (cluster-wide operator) or into one Role per watched namespace
(operator.watchNamespaces).
*/}}
{{- define "kafscale.operatorRules" -}}
- apiGroups: [""]
  resources: ["pods", "services", "endpoints", "configmaps", "secrets", "events"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
- apiGroups: ["kafscale.io"]
  resources: ["kafscaleclusters", "kafscaleclusters/status", "kafscaletopics", "kafscaletopics/status"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
- apiGroups: ["apps"]
  resources: ["deployments", "statefulsets", "daemonsets", "replicasets"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
- apiGroups: ["autoscaling"]
  resources: ["horizontalpodautoscalers"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
- apiGroups: ["batch"]
  resources: ["jobs", "cronjobs"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
- apiGroups: ["networking.k8s.io"]
  resources: ["ingresses"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
- apiGroups: ["policy"]
  resources: ["poddisruptionbudgets"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
- apiGroups: ["coordination.k8s.io"]
  resources: ["leases"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
{{- end }}

{{/*
Permissions for leader election, needed in the release namespace.
*/}}
{{- define "kafscale.operatorLeaderElectionRules" -}}
- apiGroups: ["coordination.k8s.io"]
  resources: ["leases"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
- apiGroups: [""]
  resources: ["events"]
  verbs: ["create", "patch"]
{{- end }}
