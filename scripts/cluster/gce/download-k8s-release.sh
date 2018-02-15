#!/bin/bash

set -euo pipefail

source $(dirname $0)/gce-common.sh

# When upgrading the Kubernetes release, please
# verify that the end-to-end tests work after the upgrade.
KUBERNETES_RELEASE="v1.9.1"

KUBERNETES_SKIP_CONFIRM=1
KUBERNETES_SKIP_CREATE_CLUSTER=1

cd ${STOLOS_TMP}
export KUBERNETES_RELEASE KUBERNETES_SKIP_CONFIRM KUBERNETES_SKIP_CREATE_CLUSTER
if [[ -f ./kubernetes/version && "${KUBERNETES_RELEASE}" == "$(cat ./kubernetes/version)" ]]; then
  echo "Already have up to date kubernetes, skipping download"
  exit
fi
curl -sS https://get.k8s.io | bash
