#!/bin/bash

set -euv

# Creates secret and configmap required by GitPolicyImporter.
#
# Current context should be set to the cluster running
# GitPolicyImporter.

# Remove existing resources.
kubectl delete secret git-creds -n stolos-system || true
kubectl delete configmap git-policy-importer -n stolos-system || true

kubectl create secret generic git-creds -n stolos-system \
  --from-file=ssh=${HOME}/.ssh/foo_corp_rsa  \
  --from-file=known_hosts=${HOME}/.ssh/known_hosts
kubectl create configmap git-policy-importer -n stolos-system --from-env-file=git-importer.env
