#!/bin/bash

# This script reverses the changes made by the 'patch-apiserver.sh' script.
#
# TODO(fmil): I observed that the cluster does not quite revert into the
# previous state when unpatched and API server is restarted.  Instead, various
# system bits stop working.  This manifests for example as inability to schedule
# new jobs (nodes become NotReady), and inability to terminate a pod (it remains
# in state 'Terminating' indefinitely).  I don't expect that this will remain
# the case when we get a full-fledged deployment, but is the case for now.

set -v

PROJECT_ID=${PROJECT_ID:-fmil-k8s-test}
INSTANCE_ID=${INSTANCE_ID:-kubernetes-master}
ZONE=${ZONE:-us-central1-b}

gcloud compute --project="${PROJECT_ID}" ssh --zone="${ZONE}" "${INSTANCE_ID}" \
  -- "sudo cp kube-apiserver.manifest.orig /etc/kubernetes/manifests/kube-apiserver.manifest"
