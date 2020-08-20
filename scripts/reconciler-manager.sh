#!/bin/bash

set -euo pipefail

# Pre-requisite:
#
# Gcloud: stolos-dev.
# kubectl: user cluster in Stolos-dev.

# Setup

# shellcheck disable=SC2086
docker build -t gcr.io/stolos-dev/${USER}/reconciler:latest . -f build/reconciler-manager/Dockerfile
# shellcheck disable=SC2086
docker push gcr.io/stolos-dev/${USER}/reconciler:latest

# Install CRDs for successfully running importer pod.
# Note: CRD installation will handled by ConfigSync operator in future.
kubectl apply -f manifests/00-namespace.yaml
kubectl apply -f manifests/cluster-config-crd.yaml
kubectl apply -f manifests/cluster-config-crd.yaml
kubectl apply -f manifests/sync-declaration-crd.yaml
kubectl apply -f manifests/nomos-repo-crd.yaml
kubectl apply -f manifests/namespace-selector-crd.yaml
kubectl apply -f manifests/namespace-config-crd.yaml
kubectl apply -f manifests/hierarchyconfig-crd.yaml
kubectl apply -f manifests/cluster-selector-crd.yaml
kubectl apply -f manifests/cluster-registry-crd.yaml
kubectl apply -f manifests/reconciler-manager-service-account.yaml
kubectl apply -f manifests/reconciler-manager-deployment-configmap.yaml

# Install the CRD's for RootSync and RepoSync types (Verify after installation).
kubectl apply -f manifests/reposync-crd.yaml
kubectl apply -f manifests/rootsync-crd.yaml

# verify if configmanagement.gke.io resources exists with RootSync and RepoSync KIND.

rm -rf /tmp/reconciler-manager.yaml
# shellcheck disable=SC2046
sed -e 's,RMUSER,'$(whoami)',g' ./manifests/templates/reconciler-manager/reconciler-manager.yaml > /tmp/reconciler-manager.yaml

# Apply the reconciler-manager.yaml manifest.
kubectl apply -f /tmp/reconciler-manager.yaml
kubectl apply -f ./manifests/templates/reconciler-manager/dev.yaml

# Cleanup exiting git-creds secret.
kubectl delete secret git-creds -n=config-management-system --ignore-not-found
# Create git-creds secret.
# shellcheck disable=SC2086
kubectl create secret generic git-creds -n=config-management-system \
      --from-file=ssh=${HOME}/.ssh/id_rsa.nomos

# Verify reconciler-manager pod is running.

# Apply RootSync CR.
#kubectl apply -f e2e/testdata/reconciler-manager/rootsync-sample.yaml

# Apply RepoSync CR.
kubectl apply -f e2e/testdata/reconciler-manager/reposync-sample.yaml

sleep 10s

# Verify whether respective controllers create various obejcts i.e. Deployments.
kubectl get all -n config-management-system

# Verify whether config map is created.
kubectl get cm -n config-management-system

# End
