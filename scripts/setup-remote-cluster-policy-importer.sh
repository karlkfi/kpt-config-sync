#!/bin/bash

set -euo pipefail

# Creates secret and configmap required by RemoteClusterPolicyImporter.
#
# Current context should be set to the cluster running
# RemoteClusterPolicyImporter.
#
# Usage: script SA_NAME REMOTE_CONTEXT
#
# Arguments:
#
# Service account name to create on remote cluster
SA_NAME=$1
# Kubeconfig context pointing to remote cluster.
REMOTE_CONTEXT=$2

# Remove existing resources on local and remote clusters
kubectl delete sa ${SA_NAME} -n stolos-system --context ${REMOTE_CONTEXT} || true
kubectl delete clusterrolebinding stolos-policy-reader-${SA_NAME} \
  -n stolos-system --context ${REMOTE_CONTEXT} || true
kubectl delete secret remote-cluster-policy-importer-master-secret -n stolos-system || true
kubectl delete configmap remote-cluster-policy-importer -n stolos-system || true

# Create a service account on the remote cluster and give it policy-reader
# permissions.
kubectl create sa ${SA_NAME} -n stolos-system --context ${REMOTE_CONTEXT}
kubectl create clusterrolebinding stolos-policy-reader-${SA_NAME} \
  --clusterrole=stolos-policy-reader --user=system:serviceaccount:stolos-system:${SA_NAME} \
  --context ${REMOTE_CONTEXT}

# Create a secret on the local cluster using the token and cert from the
# service account secret on the remote cluster.
secret_name=$(kubectl get sa ${SA_NAME} -n stolos-system --context ${REMOTE_CONTEXT} \
  -o jsonpath='{.secrets[].name}')
token=$(kubectl get secret ${secret_name} -n stolos-system --context ${REMOTE_CONTEXT} \
  -o jsonpath='{.data.token}' | base64 -d)
ca_cert=$(kubectl get secret ${secret_name} -n stolos-system --context ${REMOTE_CONTEXT} \
  -o jsonpath='{.data.ca\.crt}' | base64 -d)
[[ ! -z ${token}  ]] || { echo "failed to get token"; exit 1; }
[[ ! -z ${ca_cert}  ]] || { echo "failed to get cert"; exit 1; }
kubectl create secret generic remote-cluster-policy-importer-master-secret -n stolos-system \
  --from-literal=token="${token}" \
  --from-literal=ca.crt="${ca_cert}"

# Create configmap containing the URL of remote cluster API server.
remote_cluster=$(kubectl config view \
  -o jsonpath="{.contexts[?(@.name == \"${REMOTE_CONTEXT}\")].context.cluster}")
remote_cluster_url=$(kubectl config view \
  -o jsonpath="{.clusters[?(@.name == \"${remote_cluster}\")].cluster.server}")
[[ ! -z ${remote_cluster_url}  ]] || { echo "failed to get cluster url"; exit 1; }
kubectl create configmap remote-cluster-policy-importer -n stolos-system \
  --from-literal=master.endpoint="${remote_cluster_url}"
