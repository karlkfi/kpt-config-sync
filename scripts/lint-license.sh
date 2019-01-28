#!/bin/bash
#
# Copyright 2018 Nomos Authors
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

echo "Running licenselinter: "
"${GOPATH}/bin/linux_amd64/licenselinter" -dir "$(pwd)" -print-aggregate > /tmp/LICENSE
echo "PASS"
echo

echo "Detecting LICENSE changes: "
diff -q /tmp/LICENSE LICENSE || (echo "LICENSE file has changed, commit the changes: 'cp .output/tmp/LICENSE LICENSE'" && exit 1)
echo "PASS"
echo
