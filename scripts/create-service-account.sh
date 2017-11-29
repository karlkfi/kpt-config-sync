#!/bin/bash
#
# Copyright 2017 Kubernetes Authors
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
#
# Creates a service account if it does not already exist.
#

set -euo pipefail

account_name=${1:-}
if [[ "${account_name}" == "" ]]; then
  echo "Specify account name"
  exit 1
fi
if ! kubectl get sa/${account_name} &> /dev/null; then
  kubectl create sa ${account_name}
else
  echo "Account ${account_name} exists, will not create"
fi
