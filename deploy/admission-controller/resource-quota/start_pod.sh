#!/bin/bash
#
# Deletes if needed, and then creates the pod running the resource quota admission controller

YAML=pod.yaml
POD=admit-resource-quota

if kubectl get pod ${POD} &> /dev/null || kubectl get svc ${POD} &> /dev/null; then
echo "Found existing pod, deleting..."
kubectl delete -f ${YAML}
echo -n "Waiting for pod to terminate..."
while kubectl get pod ${POD} &> /dev/null || kubectl get svc ${POD} &> /dev/null; do
  echo -n "."
  sleep 1
done
echo "Pod terminated"
fi
kubectl create -f ${YAML}
