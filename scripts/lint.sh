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

echo "Running gometalinter: "
gometalinter.v2 \
    --deadline=60s \
    --disable-all \
    --enable=vet \
    --enable=goimports \
    --enable=deadcode \
    --enable=ineffassign \
    --enable=vetshadow \
    --enable=unconvert \
    --enable=errcheck \
    --enable=golint \
    --enable=megacheck \
    --exclude=generated\.pb\.go \
    --exclude=generated\.go \
    "$@"
echo "PASS"


echo "Running licenselinter: "
licenselinter -dir $(pwd)
echo "PASS"
echo
