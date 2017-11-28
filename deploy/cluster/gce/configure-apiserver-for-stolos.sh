#!/bin/bash

set -euv

INSTANCE_ID=${INSTANCE_ID:-${USER}-master}
API_SERVER_POD_NAME=kube-apiserver-${INSTANCE_ID}

# Generate the api-server manifest
APISERVER_IMAGE_NAME=$(kubectl get pods --namespace=kube-system ${API_SERVER_POD_NAME} -o jsonpath="{.spec.containers[0].image}")
APISERVER_IP_ADDRESS=$(kubectl get endpoints kubernetes -o jsonpath="{.subsets[0].addresses[0].ip}")
m4 \
  -DAPISERVER_IMAGE_NAME=${APISERVER_IMAGE_NAME} \
  -DAPISERVER_IP_ADDRESS=${APISERVER_IP_ADDRESS} \
  < kube-apiserver.manifest.orig.m4 \
  > kube-apiserver.manifest.orig
m4 \
  -DAPISERVER_IMAGE_NAME=${APISERVER_IMAGE_NAME} \
  -DAPISERVER_IP_ADDRESS=${APISERVER_IP_ADDRESS} \
  < kube-apiserver.manifest.patched.m4 \
  > kube-apiserver.manifest.patched

# Copies the custom configuration over to GCE.
gcloud compute scp \
  authz_kubeconfig.yaml \
  install.sh \
  kube-apiserver.manifest.patched \
  kube-apiserver.manifest.orig \
  ${USER}@${INSTANCE_ID}:~/
gcloud compute scp \
  $HOME/.minikube/ca.crt \
  ${USER}@${INSTANCE_ID}:~/ca-webhook.crt

# Calling the 'install.sh' script
# will cause a restart of the kube-apiserver on the master.
gcloud compute ssh "${INSTANCE_ID}" \
  -- "chmod ugo+rx install.sh"
gcloud compute ssh "${INSTANCE_ID}" \
  -- "sudo /bin/bash ./install.sh"
