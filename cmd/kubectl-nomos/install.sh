#!/bin/bash

set -euo pipefail

go install github.com/google/nomos/cmd/kubectl-nomos

GOPATH=${GOPATH:-${HOME}/go}
PLUGIN_DIR=$HOME/.kube/plugins/nomos
mkdir -p ${PLUGIN_DIR}
cp $(echo ${GOPATH} | sed -e 's/.*://')/bin/kubectl-nomos ${PLUGIN_DIR}

cp plugin.yaml ${PLUGIN_DIR}

echo "To test, run:"
echo "kubectl plugin nomos"
