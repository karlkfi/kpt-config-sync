#!/bin/bash

# Copyright 2018 The Nomos Authors.
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

# This script initializes the test git server used during e2e tests.
# In particular, it configures the ssh key for accessing the service using
# the local public key, sets up ssh forwarding through localhost:2222 to
# to the git server to enable later changing of the contents of the hosted
# repo, and initializes the git repo to use.
#
# IMPORTANT NOTE: this script makes use of ~/.ssh/id_rsa.pub, so this must
# be created for the script to work properly

TEST_LOG_REPO=/tmp/nomos-test

FWD_SSH_PORT=2222

kubectl -n=nomos-system-test create secret generic ssh-key-secret --from-file=${HOME}/.ssh/id_rsa.pub
until kubectl get pods -n=nomos-system-test --field-selector=status.phase=Running -lapp=test-git-server | grep -qe Running; do
   sleep 1;
   echo "Waiting for test-git-server pod to be ready..."
done

echo "test-git-server ready"

POD_ID=$(kubectl get pods -n=nomos-system-test -l app=test-git-server -o jsonpath='{.items[0].metadata.name}')

mkdir -p ${TEST_LOG_REPO}
kubectl -n=nomos-system-test port-forward ${POD_ID} ${FWD_SSH_PORT}:22 > ${TEST_LOG_REPO}/port-forward.log &
kubectl exec -n=nomos-system-test -it ${POD_ID} -- git init --bare --shared /git-server/repos/sot.git
