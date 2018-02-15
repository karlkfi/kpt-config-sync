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
# controller.  Intended to be invoked from the top-level directory.
#
# Invoke as:
#   deploy-resourcequota-admission-controller.sh \
#     staging_dir gen_yaml_dir

set -euo pipefail

STAGING_DIR=$1
GEN_YAML_DIR=$2

echo STAGING_DIR=${STAGING_DIR}
echo GEN_YAML_DIR=${GEN_YAML_DIR}

cd ${STAGING_DIR}/resourcequota-admission-controller; ./gencert.sh
kubectl delete secret resourcequota-admission-controller-secret \
    --namespace=stolos-system || true
kubectl create secret tls resourcequota-admission-controller-secret \
    --namespace=stolos-system \
    --cert=${STAGING_DIR}/resourcequota-admission-controller/server.crt \
    --key=${STAGING_DIR}/resourcequota-admission-controller/server.key
kubectl delete secret resourcequota-admission-controller-secret-ca \
    --namespace=stolos-system || true
kubectl create secret tls resourcequota-admission-controller-secret-ca \
    --namespace=stolos-system \
    --cert=${STAGING_DIR}/resourcequota-admission-controller/ca.crt \
    --key=${STAGING_DIR}/resourcequota-admission-controller/ca.key
kubectl delete ValidatingWebhookConfiguration stolos-resource-quota --ignore-not-found
kubectl apply -f ${GEN_YAML_DIR}/resourcequota-admission-controller.yaml

