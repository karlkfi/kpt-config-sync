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
#

# Generates the certificate used for webhooks.

set -euv

NOMOS="${NOMOS:-$(git -C "$(dirname "$0")" rev-parse --show-toplevel)}"

# Generate a CA and key for the API server.
mkdir -p "${NOMOS}/certs"

CN="nomosCA"
openssl genrsa -out "${NOMOS}/certs/ca.key" 2048
openssl req -x509 -new -nodes \
  -key "${NOMOS}/certs/ca.key" \
  -days 100000 \
  -out "${NOMOS}/certs/ca.crt" \
  -subj "/CN=${CN}"


