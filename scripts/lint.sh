#!/bin/bash
#
# Copyright 2016 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -euo pipefail

export CGO_ENABLED=0

####### GOLINT FIXIT #############
#
# When you fix one of the packages here, remove it from the list.
#
GOLINT_EXCLUDE_PACKAGES=(
  cmd/syncer-demo-driver
  pkg/admissioncontroller
  pkg/admissioncontroller/policynode
  pkg/admissioncontroller/resourcequota
  pkg/api/policyhierarchy/v1
  pkg/client/action
  pkg/client/action/test
  pkg/client/restconfig
  pkg/cli/namespaces
  pkg/cli/testing
  pkg/installer
  pkg/installer/config
  pkg/policyimporter
  pkg/policyimporter/actions
  pkg/policyimporter/filesystem
  pkg/process/configgen
  pkg/process/dialog
  pkg/process/exec
  pkg/process/kubectl
  pkg/resourcequota
  pkg/service
  pkg/syncer
  pkg/syncer/clusterpolicycontroller
  pkg/syncer/clusterpolicycontroller/modules
  pkg/syncer/flattening
  pkg/syncer/hierarchy
  pkg/syncer/labeling
  pkg/syncer/policyhierarchycontroller/testing
  pkg/testing/e2e/testcontext
  pkg/testing/fakeinformers
  pkg/testing/orgdriver
  pkg/testing/rbactesting
  pkg/util/log
  pkg/util/policynode/validator
  pkg/version
)
function exclude() {
  local value="${1:-}"
  for i in "${GOLINT_EXCLUDE_PACKAGES[@]}"; do
    if [[ "${i}" == "${value}" ]]; then
      return 0
    fi
  done
  return 1
}
LINT_PKGS=()
for pkg in $(find cmd pkg -name '*.go' \
    | sed -e 's|/[^/]*$||' \
    | sort \
    | uniq); do
  if ! exclude ${pkg}; then
    LINT_PKGS+=(${pkg})
  fi
done



echo "Checking golint: "
gometalinter.v2 \
    --disable-all \
    --enable=golint \
    "${LINT_PKGS[@]}"
echo "PASS"

echo "Checking linters: "
gometalinter.v2 \
    --disable-all \
    --enable=vet \
    --enable=goimports \
    --enable=deadcode \
    --enable=ineffassign \
    --enable=vetshadow \
    --enable=unconvert \
    --enable=errcheck \
    --exclude=generated\.pb\.go \
    "$@"
echo "PASS"
echo
