#!/bin/bash

set -euo pipefail

# shellcheck source=/dev/null
source "$(dirname "$0")"/gce-common.sh

# When upgrading the Kubernetes release, please
# verify that the end-to-end tests work after the upgrade.
export KUBERNETES_RELEASE="v1.9.1"
export KUBERNETES_SKIP_CONFIRM=1
export KUBERNETES_SKIP_CREATE_CLUSTER=1

(
  cd "${NOMOS_TMP}"
  if [[ -f ./kubernetes/version && "${KUBERNETES_RELEASE}" == "$(cat ./kubernetes/version)" ]]; then
    echo "Already have up to date kubernetes, skipping download"
    exit
  fi
  (curl -sS https://get.k8s.io | bash)
)

echo "Injecting admission controllers hook"
PATCHFILE="$(dirname "$0")"/admission-control.patch-"${KUBERNETES_RELEASE}"
patch -p0 -N \
  "${NOMOS_TMP}/kubernetes/cluster/gce/config-default.sh" \
  < "${PATCHFILE}"
