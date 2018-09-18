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
#
# Setup script for installing pre-requisites before running other scripts in
# deploy.  Things in this file should generally follow the lifecycle of the
# kubernetes cluster and only be created once.

# Generates the following certificates in ${controller_staging_dir}:
#
# - ca.key: the private key for a self-signed certificate authority.
# - ca.crt: the certificate corresponding to ca.key.
# - server.key: the private key for the server that we are generating certs for.
# - server.crt: the certificate signed by ca.key for the server.
#
# The server certificate will include the subject CN=<name of
# service> to allow the admission controller to be called by the api server.
#
# Parameters:
#    OUTPUT_DIR: define this environment variable to set the output directory
#       for the generated keys and certificates.  Otherwise, the output staging
#       directory for the resourcequota-admission-controller is assumed.

set -euo pipefail

readonly controller_staging_dir="${OUTPUT_DIR:-./resourcequota-admission-controller}"

# The name of the service (and the namespace it runs in).
readonly cn_base="resourcequota-admission-controller.nomos-system.svc"

"$(dirname "$0")"/generate-admission-controller-certs.sh \
  --domain "${cn_base}" \
  --output_dir "${controller_staging_dir}"
