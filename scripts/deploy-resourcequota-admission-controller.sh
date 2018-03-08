#!/bin/bash
#
# Copyright 2018 Stolos Authors
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
#

# Creates the required secrets for the resourcequota admission
# controller.
#
# Parameters:
#    YAML_DIR: Define this environment variable to point to the YAML files for
#        starting the resourcequota admission controller. Required.
#    CERTS_INPUT_DIR: Define this environment variable to point to the directory
#        that contains server.crt, server.key, ca.crt, ca.key.  Otherwise,
#        define the following:
#
#        - SERVER_CERT_FILE: the path to server.crt
#        - SERVER_KEY_FILE: the path to server.key
#        - CA_CERT_FILE: the path to ca.crt
#        - CA_KEY_FILE: the path to ca.key

set -euo pipefail

echo CERTS_INPUT_DIR=${CERTS_INPUT_DIR:-""}
echo YAML_DIR=$YAML_DIR

readonly server_cert_file="${SERVER_CERT_FILE:-${CERTS_INPUT_DIR}/server.crt}"
readonly server_key_file="${SERVER_KEY_FILE:-${CERTS_INPUT_DIR}/server.key}"
readonly ca_key_file="${CA_KEY_FILE:-${CERTS_INPUT_DIR}/ca.key}"
readonly ca_cert_file="${CA_CERT_FILE:-${CERTS_INPUT_DIR}/ca.crt}"

kubectl delete secret resourcequota-admission-controller-secret \
    --namespace=stolos-system || true
kubectl create secret tls resourcequota-admission-controller-secret \
    --namespace=stolos-system \
    --cert="${server_cert_file}" \
    --key="${server_key_file}"

kubectl delete secret resourcequota-admission-controller-secret-ca \
    --namespace=stolos-system || true
kubectl create secret tls resourcequota-admission-controller-secret-ca \
    --namespace=stolos-system \
    --cert=${ca_cert_file} \
    --key=${ca_key_file}
kubectl delete ValidatingWebhookConfiguration stolos-resource-quota \
    --ignore-not-found

kubectl apply -f ${YAML_DIR}/resourcequota-admission-controller.yaml

