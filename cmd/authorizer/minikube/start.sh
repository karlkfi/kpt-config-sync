#!/bin/bash
# Starts minikube with the flag settings required for webhook bootstrap.

set -v

minikube status

minikube start \
  --extra-config="apiserver.Authorization.WebhookConfigFile=/etc/kubernetes/addons/webhook.kubeconfig" \
  --extra-config="apiserver.Authorization.Mode=Webhook,AlwaysAllow" \
  --logtostderr \
  "$@"

eval $(minikube docker-env)

MINIKUBE_HOST_ADDRESS=$(minikube ip)
echo "Minkube cluster IP: ${MINIKUBE_HOST_ADDRESS}"

echo "Deploying authorizer POD:"
kubectl replace --force -f minikube/authorizer_deploy.yaml


