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
set -x

STAGING_DIR="${STAGING_DIR:-$(dirname ${0:-''})/../.output}/staging"
readonly controller_staging_dir="${OUTPUT_DIR:-${STAGING_DIR}/resourcequota-admission-controller}"

# The name of the service (and the namespace it runs in).
readonly cn_base="resourcequota-admission-controller.nomos-system.svc"

mkdir -p "${controller_staging_dir}"
cd "${controller_staging_dir}"

# Create a certificate authority
openssl genrsa -out ca.key 2048
openssl req -x509 -new -nodes -key ca.key -days 100000 \
  -out ca.crt -subj "/CN=${cn_base}_ca"

# Create a server certificate
readonly tempfile=$(mktemp --tmpdir=${PWD} gencerts-server-cert-XXXXXX.conf)
echo "Using config file: ${tempfile}"
cat >"${tempfile}" <<EOF
[req]
req_extensions = v3_req
distinguished_name = req_distinguished_name
[req_distinguished_name]
[ v3_req ]
basicConstraints = CA:FALSE
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
extendedKeyUsage = clientAuth, serverAuth
EOF


openssl genrsa -out server.key 2048
openssl req -new -key server.key -out server.csr \
  -subj "/CN=${cn_base}" -config "${tempfile}"
openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key \
  -CAcreateserial -out server.crt -days 100000 \
  -extensions v3_req -extfile "${tempfile}"

rm "${tempfile}"

