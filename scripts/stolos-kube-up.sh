#!/bin/bash
# Download and run a patched version of Kubernetes for a Stolos pre-release.
# This version has been patched to include support for self-hosting webhook
# authorizer, which as of this writing (2017-10-20) has not yet been submitted
# to the K8S code base.

readonly K8S_BASE_DIR=${K8S_BASE_DIR:-$HOME/local/opt/k8s}

export NUM_NODES={$NUM_NODES:=1}
export KUBERNETES_RELEASE_URL="https://storage.googleapis.com/stolos/k8s"
export KUBERNETES_CI_RELEASE_URL=${KUBERNETES_RELEASE_URL}
export KUBERNETES_RELEASE={$KUBERNETES_RELEASE:-"v1.8.0"}

cd $K8S_BASE_DIR
./download.sh

