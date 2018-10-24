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

# The linters run for a long time and even longer on shared machines that run
# our presubmit tests.  Allow a generous timeout for the linter checks to
# finish.
export linter_deadline="500s"

echo "Running gometalinter: "
if ! OUT="$(
  gometalinter.v2 \
    --deadline="${linter_deadline}" \
    --disable-all \
    --enable=deadcode \
    --enable=errcheck \
    --enable=golint \
    --enable=ineffassign \
    --enable=megacheck \
    --enable=unconvert \
    --enable=vet \
    --enable=vetshadow \
    --exclude=generated\.pb\.go \
    --exclude=generated\.go \
		--exclude=zz_generated\.*\.go \
    "$@" \
    )"; then
  echo "${OUT}"

  NC=''
  RED=''
  if [ -t 1 ]; then
    NC='\033[0m'
    RED='\033[0;31m'
  fi

  if echo "${OUT}" | grep "(goimports)" > /dev/null; then
    echo -e "${RED}ADVICE${NC}: runing \"make goimports\" may fix the (goimports) error"
  fi
  exit 1
fi
echo "PASS"
echo
