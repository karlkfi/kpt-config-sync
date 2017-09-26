#!/bin/bash

set -euo pipefail

# Default image name
IMAGE=syncer:test

function finish() {
  kubectl config use-context ${kubectl_context}
}

function restore_context_on_exit() {
  kubectl_context=$(kubectl config current-context)
  echo "Saving context ${kubectl_context}"
  trap finish EXIT
}

function build_image() {
  make clean
  make
}

# Generate the yaml file and create a new pod
function create_pod() {
  local yaml_file=syncer-pod-${1:-}.yaml
  local image_name="${2:-}"

  m4 -DIMAGE_NAME=${image_name} < tpl/syncer-pod.yaml > ${yaml_file}

  if kubectl get -f ${yaml_file} &> /dev/null; then
    echo "Found existing pod, deleting..."
    kubectl delete -f ${yaml_file}
    echo -n "Waiting for pod to terminate..."
    while kubectl get -f ${yaml_file} &> /dev/null; do
      echo -n "."
      sleep 1
    done
    echo "Pod terminated"
  fi
  kubectl create -f ${yaml_file}
}
