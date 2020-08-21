#!/bin/bash

set -euo pipefail

# Pre-requisite:
#
# Gcloud: stolos-dev.
# kubectl: user cluster in Stolos-dev.

# Setup

MGR_TAG="gcr.io/stolos-dev/${USER}/reconciler-manager:latest"
REC_TAG="gcr.io/stolos-dev/${USER}/reconciler:latest"

docker build -t "${MGR_TAG}" . -f build/reconciler-manager/Dockerfile
docker build -t "${REC_TAG}" . -f build/reconciler/Dockerfile
docker push "${MGR_TAG}"
docker push "${REC_TAG}"

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

# Fill in configmap template
sed -e "s|IMAGE_NAME|$REC_TAG|" \
  manifests/templates/reconciler-manager-deployment-configmap.yaml \
  | tee reconciler-manager-deployment-configmap.generated.yaml \
  | kubectl apply -f -

# Install the CRD's for RootSync and RepoSync types (Verify after installation).
kubectl apply -f manifests/reposync-crd.yaml
kubectl apply -f manifests/rootsync-crd.yaml

# verify if configmanagement.gke.io resources exists with RootSync and RepoSync KIND.
# Apply the reconciler-manager.yaml manifest.
sed -e "s|IMAGE_NAME|$MGR_TAG|g" ./manifests/templates/reconciler-manager.yaml \
  | tee reconciler-manager.generated.yaml \
  | kubectl apply -f -
kubectl apply -f ./manifests/templates/reconciler-manager/dev.yaml

# Cleanup exiting root-ssh-key secret.
kubectl delete secret root-ssh-key -n=config-management-system --ignore-not-found
# Create root-ssh-key secret for Root Reconciler.
# shellcheck disable=SC2086
kubectl create secret generic root-ssh-key -n=config-management-system \
      --from-file=ssh=${HOME}/.ssh/id_rsa.nomos

# Cleanup exiting ssh-key secret.
kubectl delete secret ssh-key -n=bookinfo --ignore-not-found
# Create ssh-key secret for Namespace Reconciler.
# shellcheck disable=SC2086
kubectl create secret generic ssh-key -n=bookinfo \
      --from-file=ssh=${HOME}/.ssh/id_rsa.nomos

# Verify reconciler-manager pod is running.

# Apply RootSync CR.
kubectl apply -f e2e/testdata/reconciler-manager/rootsync-sample.yaml

# Apply RepoSync CR.
kubectl apply -f e2e/testdata/reconciler-manager/reposync-sample.yaml

sleep 10s

# Verify whether respective controllers create various obejcts i.e. Deployments.
kubectl get all -n config-management-system

# Verify whether config map is created.
kubectl get cm -n config-management-system

# End
