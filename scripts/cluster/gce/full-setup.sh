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

# One-time setup for a GCE cluster that needs to run Nomos.

set -euo pipefail

echo "Setting project to stolos-dev"
gcloud config set project stolos-dev
echo "Setting compute zone to us-central1-b"
gcloud config set compute/zone us-central1-b

echo "Generating certificates"
./gencerts.sh

echo "Downloading kubernetes release"
./download-k8s-release.sh

echo "Setting up base kubernetes cluster"
./kube-up.sh

echo "Configuring monitoring"
./configure-monitoring.sh
