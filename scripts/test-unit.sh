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

export PATH="${PATH}:./third_party/bats-core/libexec"

echo "Running go tests:"
go test -i -installsuffix "static" "$@"
go test -installsuffix "static" "$@"
echo "PASS"

# Keep BATS tests behind go tests.  bats seems to pollute the environment, and
# go doesn't like that.
echo
echo "Running BATS tests:"
for testfile in "$(find ./scripts -name '*.bats')"; do
  . ./third_party/bats-core/bin/bats "${testfile}"
done
echo "PASS"

echo
