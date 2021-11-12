#!/bin/bash
set -o errexit
set -o nounset
set -o pipefail

if ! [ -x "$(command -v golangci-lint)" ]; then
  echo 'golangci-lint is not installed.'
  echo 'installing golangci-lint'
  curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.24.0
  echo 'finished installing golangci-lint'
fi

make lint
make test