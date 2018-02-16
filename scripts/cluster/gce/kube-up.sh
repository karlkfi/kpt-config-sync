#!/bin/bash

set -euo pipefail

source $(dirname $0)/gce-common.sh

if kubectl config use-context stolos-dev_${KUBE_GCE_INSTANCE_PREFIX} &> /dev/null; then
  if kubectl get ns &> /dev/null; then
    echo "Cluster already created"
    exit
  fi
fi

gcloud config get-value project
${STOLOS_TMP}/kubernetes/cluster/kube-up.sh
