#!/usr/bin/env bash

set -euo pipefail

# Install helm server (tiller)
kubectl create sa tiller -n kube-system
helm init --service-account=tiller
kubectl rollout status -w deployment/tiller-deploy -n kube-system

# Temporarily give tiller cluster-admin privilege
cat <<EOF | kubectl create -f -
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRoleBinding
metadata:
  name: tiller-binding
subjects:
  - kind: ServiceAccount
    name: tiller
    namespace: kube-system
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluster-admin
EOF

# Use helm to install prometheus
kubectl create ns monitoring
helm  install coreos/prometheus-operator --name prometheus-operator --namespace monitoring
helm  install coreos/kube-prometheus --name kube-prometheus --set rbacEnable=true --namespace monitoring

# Revoke tiller's privilege
kubectl delete clusterrolebinding tiller-binding
