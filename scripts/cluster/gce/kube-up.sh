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

source $(dirname $0)/gce-common.sh

if kubectl config use-context stolos-dev_${KUBE_GCE_INSTANCE_PREFIX} &> /dev/null; then
  if kubectl get ns &> /dev/null; then
    echo "Cluster already created"
    exit
  fi
fi

gcloud config get-value project
export NOMOS_ADMISSION_CONTROL="ValidatingAdmissionWebhook"
${NOMOS_TMP}/kubernetes/cluster/kube-up.sh
