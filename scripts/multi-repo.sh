#!/bin/bash

set -euo pipefail

# Pre-requisite:
#
# Gcloud: stolos-dev.
# kubectl: user cluster in Stolos-dev.

# Setup

REPO_DIR="$(readlink -f "$(dirname "$0")")/.."

GCP_PROJECT=${GCP_PROJECT:-stolos-dev}
NOMOS_TAG="gcr.io/${GCP_PROJECT}/${USER}/nomos:latest"
MGR_TAG="gcr.io/${GCP_PROJECT}/${USER}/reconciler-manager:latest"
HC_TAG="gcr.io/${GCP_PROJECT}/${USER}/hydration-controller:latest"
REC_TAG="gcr.io/${GCP_PROJECT}/${USER}/reconciler:latest"
WEBHOOK_TAG="gcr.io/${GCP_PROJECT}/${USER}/admission-webhook:latest"

make -C "${REPO_DIR}" image-nomos \
  NOMOS_TAG="${NOMOS_TAG}" \
  RECONCILER_TAG="${REC_TAG}" \
  HYDRATION_CONTROLLER_TAG="${HC_TAG}" \
  RECONCILER_MANAGER_TAG="${MGR_TAG}" \
  ADMISSION_WEBHOOK_TAG="${WEBHOOK_TAG}"
docker push "${MGR_TAG}"
docker push "${HC_TAG}"
docker push "${REC_TAG}"
docker push "${WEBHOOK_TAG}"

# Install CRDs for successfully running importer pod.
# Note: CRD installation will handled by ConfigSync operator in future.
kubectl apply -f manifests/test-resources/00-namespace.yaml
kubectl apply -f manifests/namespace-selector-crd.yaml
kubectl apply -f manifests/cluster-selector-crd.yaml
kubectl apply -f manifests/cluster-registry-crd.yaml
kubectl apply -f manifests/reconciler-manager-service-account.yaml
kubectl apply -f manifests/otel-agent-cm.yaml
kubectl apply -f manifests/test-resources/resourcegroup-manifest.yaml

# Fill in configmap template
sed -e "s|RECONCILER_IMAGE_NAME|$REC_TAG|" -e "s|HYDRATION_CONTROLLER_IMAGE_NAME|$HC_TAG|" \
  manifests/templates/reconciler-manager-configmap.yaml \
  | tee reconciler-manager-configmap.generated.yaml \
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

# Apply the admission-webhook.yaml manifest.
sed -e "s|IMAGE_NAME|$WEBHOOK_TAG|g" ./manifests/templates/admission-webhook.yaml \
  | tee admission-webhook.generated.yaml \
  | kubectl apply -f -

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

# Apply namespace reconciler ClusterRole.
kubectl apply -f manifests/ns-reconciler-cluster-role.yaml

# Verify reconciler-manager pod is running.

# Apply RootSync CR.
kubectl apply -f e2e/testdata/reconciler-manager/rootsync-sample.yaml

# Apply RepoSync CR.
kubectl apply -f e2e/testdata/reconciler-manager/reposync-sample.yaml

sleep 10s

# Verify whether respective controllers create various objects i.e. Deployments.
kubectl get all -n config-management-system

# Verify whether config map is created.
kubectl get cm -n config-management-system

# End
