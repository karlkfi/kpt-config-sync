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
# Depoy script for GKE.  Hopefully this will get unified with deploy-minikube
# sooner than later.
#

source $(dirname ${0:-})/common.sh

restore_context_on_exit

PROJECT_ID=$(gcloud config get-value project)
CONTEXT=""
while [[ $# -gt 0 ]]; do
  arg="${1:-}"
  case "${arg}" in
    -p|--project)
      PROJECT_ID="${2:-}"
      shift
      shift
      ;;

    -c|--context)
      CONTEXT="${2:-}"
      shift
      shift
      ;;

    *)
      break
      ;;
  esac
done

if [[ "${PROJECT_ID}" == "" ]]; then
  echo "Speicfy project id -p|--project <id>"
  exit 1
fi
if [[ "${CONTEXT}" != "" ]]; then
  kubectl config use-context ${CONTEXT}
fi

build_image

# tag docker image then push to GCP
GCP_IMAGE=us.gcr.io/${PROJECT_ID}/${IMAGE}
docker tag ${IMAGE} ${GCP_IMAGE}
gcloud docker -- push ${GCP_IMAGE}

create_pod gke ${GCP_IMAGE}
