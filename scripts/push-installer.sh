#!/bin/bash
#
# Copyright 2018 The Stolos Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License"};
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

function main() {
  echo
  echo "******* Pushing installer *******"
  echo
  cp -R ${TOP_DIR}/build/installer ${STAGING_DIR}
  cp ${BIN_DIR}/${ARCH}/installer ${STAGING_DIR}/installer
  cp ${TOP_DIR}/toolkit/installer/README.md ${STAGING_DIR}/installer

  mkdir -p ${STAGING_DIR}/installer/yaml
  cp ${OUTPUT_DIR}/yaml/*  ${STAGING_DIR}/installer/yaml

  mkdir -p ${STAGING_DIR}/installer/manifests/enrolled
  cp -R ${TOP_DIR}/manifests/enrolled/* ${STAGING_DIR}/installer/manifests/enrolled

  mkdir -p ${STAGING_DIR}/installer/manifests/common
  cp -R ${TOP_DIR}/manifests/common/* ${STAGING_DIR}/installer/manifests/common

  mkdir -p ${STAGING_DIR}/installer/configs
  cp -R ${TOP_DIR}/toolkit/installer/configs/* ${STAGING_DIR}/installer/configs
  cp ${TOP_DIR}/toolkit/installer/entrypoint.sh ${STAGING_DIR}/installer

  mkdir -p ${STAGING_DIR}/installer/scripts
  cp ${TOP_DIR}/scripts/deploy-resourcequota-admission-controller.sh \
     ${STAGING_DIR}/installer/scripts
  cp ${TOP_DIR}/scripts/generate-resourcequota-admission-controller-certs.sh \
     ${STAGING_DIR}/installer/scripts

  docker build -t latest -t gcr.io/${GCP_PROJECT}/installer:${IMAGE_TAG} ${STAGING_DIR}/installer \
    --build-arg version=${VERSION}
  gcloud docker -- push gcr.io/${GCP_PROJECT}/installer:${IMAGE_TAG}
  gcloud docker -- tag gcr.io/${GCP_PROJECT}/installer:${IMAGE_TAG} gcr.io/${GCP_PROJECT}/installer:latest

  gsutil cp ${TOP_DIR}/scripts/run-installer.sh ${RELEASE_BUCKET}
  gsutil acl ch -r -u AllUsers:R ${RELEASE_BUCKET}/run-installer.sh
}

main "$@"
