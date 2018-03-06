#!/bin/bash

set -euo pipefail

# Creates secret and configmap required by GitPolicyImporter.
#
# Current context should be set to the cluster running
# GitPolicyImporter.

# Remove existing resources.
kubectl delete secret git-creds -n stolos-system || true
kubectl delete configmap git-policy-importer -n stolos-system || true

kubectl create secret generic git-creds -n stolos-system \
  --from-file=ssh=${HOME}/.ssh/id_rsa  \
  --from-file=known_hosts=${HOME}/.ssh/known_hosts
kubectl create configmap git-policy-importer -n stolos-system --from-env-file=$(dirname $0)/git-importer.env
