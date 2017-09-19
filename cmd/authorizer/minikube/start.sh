#!/bin/bash
# Starts minikube with the flag settings required for webhook bootstrap.

set -v

minikube status

minikube start \
  --extra-config="apiserver.Authorization.WebhookConfigFile=/etc/kubernetes/addons/webhook.kubeconfig" \
  --extra-config="apiserver.Authorization.Mode=Webhook,AlwaysAllow" \
  --logtostderr \
  "$@"

minikube ip
minikube ssh -- "sudo chmod ug+x /etc/kubernetes/addons/bootlocal.sh"
minikube ssh -- "sudo /etc/kubernetes/addons/bootlocal.sh &"

