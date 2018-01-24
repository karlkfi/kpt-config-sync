#!/bin/bash
export NUM_NODES=1
export KUBE_GCE_INSTANCE_PREFIX=${KUBE_GCE_INSTANCE_PREFIX:-${USER}}
gcloud config get-value project
./_kubernetes/kubernetes/cluster/kube-up.sh
