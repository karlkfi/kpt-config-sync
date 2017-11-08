#!/bin/bash

set -euo pipefail

go install github.com/google/stolos/cmd/kubectl-stolos

PLUGIN_DIR=$HOME/.kube/plugins/stolos
mkdir -p ${PLUGIN_DIR}
cp $(echo ${GOPATH} | sed -e 's/.*://')/bin/kubectl-stolos ${PLUGIN_DIR}

cp plugin.yaml ${PLUGIN_DIR}

echo "To test, run:"
echo "kubectl plugin stolos"
