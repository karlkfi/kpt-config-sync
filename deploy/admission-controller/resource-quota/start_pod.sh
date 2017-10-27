#!/bin/bash
#
# Deletes if needed, and then creates the pod running the resource quota admission controller

YAML=pod.yaml
POD=admit-resource-quota

if kubectl --namespace=stolos-system get pod ${POD} &> /dev/null || kubectl --namespace=stolos-system get svc ${POD} &> /dev/null; then
echo "Found existing pod, deleting..."
kubectl delete -f ${YAML}
echo -n "Waiting for pod to terminate..."
while kubectl --namespace=stolos-system get pod ${POD} &> /dev/null || kubectl --namespace=stolos-system get svc ${POD} &> /dev/null; do
  echo -n "."
  sleep 1
done
echo "Pod terminated"
fi
kubectl create -f ${YAML}
