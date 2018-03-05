#!/bin/bash
#
# Copyright 2017 Stolos Authors
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

set -euo pipefail

MANIFESTS=$(dirname ${0:-})/../manifests
ACTION=${ACTION:-create}
MODE=${MODE:-enrolled}
ENROLLED_DIRS=(${MANIFESTS}/common ${MANIFESTS}/enrolled)
MASTER_DIRS=(${MANIFESTS}/common ${MANIFESTS}/master)

case $MODE in
  enrolled)
    dirs=( "${ENROLLED_DIRS[@]}" )
    ;;
  master)
    dirs=( "${MASTER_DIRS[@]}" )
    ;;
  *)
    echo "Invalid mode ${MODE}"
    exit 1
    ;;
esac

case $ACTION in
  create)
    cmd="apply"
    ;;
  delete)
    cmd="delete"
    ;;
  *)
    echo "Invalid action ${ACTION}"
    exit 1
    ;;
esac


echo "Directories: ${dirs[@]}"
for d in "${dirs[@]}"
do
  echo "kubectl $cmd -f $d"
  kubectl $cmd -f $d
  if [[ -x setup.sh ]]; then
    ./setup.sh
  fi
done

