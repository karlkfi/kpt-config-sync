#!/bin/bash

set -euo pipefail

# Creates secret and configmap required by GitPolicyImporter.
#
# Current context should be set to the cluster running
# GitPolicyImporter.

KEY=${HOME}/.ssh/my_deployment_key
KNOWN_HOSTS=${HOME}/.ssh/known_hosts

while [[ $# -gt 0 ]]; do
  key="${1:-}"
  shift # past argument
  case $key in
    -k|--key)
    KEY="${1:-}"
    shift # past value
    ;;
    -n|--known_hosts)
    KNOWN_HOSTS="${1:-}"
    shift # past value
    ;;
  esac
done

# Remove existing resources.
kubectl delete secret git-creds -n stolos-system || true
kubectl delete configmap git-policy-importer -n stolos-system || true

kubectl create secret generic git-creds -n stolos-system \
  --from-file=ssh=${KEY}  \
  --from-file=known_hosts=${KNOWN_HOSTS}
kubectl create configmap git-policy-importer -n stolos-system --from-env-file=$(dirname $0)/git-importer.env
