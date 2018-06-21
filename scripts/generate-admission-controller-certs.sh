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
#
# This script generates certificates for an admission controller.     The server
# certificate will have the subject CN=<name of service> to allow the admission
# controller to be called by the api server.  This script generates certificates
# for an admission controller.  It will place the following files in the
# --output_dir directory:
#   - ca.key: the certificate authority's private self-signed key
#   - ca.crt: the certificate authority's ca.key
#   - server.key: the admission controller's private key
#   - server.crt: the admission controller's certificate signed by ca.key
#
# Flags:
#   --domain: the common name to put in the certs
#   --output_dir: where to place the generated certs

set -euo pipefail

domain_name=""
output_dir=""
while [[ $# -gt 0 ]]; do
  arg="${1:-}"
  shift
  case ${arg} in
    --domain)
      domain_name="${1:-}"
      shift
      ;;
    --output_dir)
      output_dir="${1:-}"
      shift
      ;;
    *)
      echo "Unknown arg $arg"
      exit 1
      ;;
  esac
done

if [[ "${domain_name}" == "" ]]; then echo "Specify --domain"; exit 1; fi
if [[ "${output_dir}" == "" ]]; then echo "Specify --output_dir"; exit 1; fi

mkdir -p "${output_dir}"
cd "${output_dir}"

echo "Creating a certificate authority for ${domain_name}"
openssl genrsa -out ca.key 2048
openssl req -x509 -new -nodes -key ca.key -days 100000 \
  -out ca.crt -subj "/CN=${domain_name}_ca"

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

echo "Generating certs for ${domain_name}"
openssl genrsa -out server.key 2048
openssl req -new -key server.key -out server.csr \
  -subj "/CN=${domain_name}" -config "${tempfile}"
openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key \
  -CAcreateserial -out server.crt -days 100000 \
  -extensions v3_req -extfile "${tempfile}"

rm "${tempfile}"
