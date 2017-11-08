#!/bin/bash

# Copies the custom configuration over to GCE.  Calling the 'install.sh' script
# will cause a restart of the kube-apiserver on the master.

set -v

echo "PWD: $PWD"

PROJECT_ID=${PROJECT_ID:-fmil-k8s-test}
INSTANCE_ID=${INSTANCE_ID:-kubernetes-master}
ZONE=${ZONE:-us-central1-b}
VERSION=${VERSION:-"1.8.0"}

gcloud compute --project="${PROJECT_ID}" scp --zone="${ZONE}" \
  authz_kubeconfig.yaml \
  install.sh \
  ${VERSION}/kube-apiserver.manifest.patched \
  ${VERSION}/kube-apiserver.manifest.orig \
  ${USER}@${INSTANCE_ID}:~/
gcloud compute --project="${PROJECT_ID}" scp --zone="${ZONE}" \
  $HOME/.minikube/ca.crt \
  ${USER}@${INSTANCE_ID}:~/ca-webhook.crt

gcloud compute --project="${PROJECT_ID}" ssh --zone="${ZONE}" "${INSTANCE_ID}" \
  -- "chmod ugo+rx install.sh"

gcloud compute --project="${PROJECT_ID}" ssh --zone="${ZONE}" "${INSTANCE_ID}" \
  -- "sudo /bin/bash ./install.sh"
