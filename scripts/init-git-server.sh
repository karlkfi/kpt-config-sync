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
# IMPORTANT NOTE: this script makes use of ~/.ssh/id_rsa.nomos.pub, so this must
# be created for the script to work properly

TEST_LOG_REPO=/tmp/nomos-test

FWD_SSH_PORT=2222

GIT_SERVER_NS=nomos-system-test

rm -rf "${TEST_LOG_REPO}"

kubectl apply -f /opt/testing/e2e/git-server.yaml

kubectl -n="${GIT_SERVER_NS}" \
  create secret generic ssh-pub \
  --from-file=/opt/testing/e2e/id_rsa.nomos.pub
echo -n "Waiting for test-git-server pod to be ready. This could take a minute..."

NEXT_WAIT_TIME=0
until kubectl get pods -n=${GIT_SERVER_NS} -lapp=test-git-server | grep -qe Running || [ $NEXT_WAIT_TIME -eq 10 ]; do
  # I've seen this take anywhere from 2 to 40 seconds, so set the polling
  # interval for reasonable granularity within that
  sleep $(( NEXT_WAIT_TIME++ ))
  echo -n "."
done

if [ $NEXT_WAIT_TIME -eq 10 ]
then
  echo "timeout waiting for test-git-server to come up expired"
  kubectl get events -n "${GIT_SERVER_NS}"
  exit 1
fi

echo "test-git-server ready"

POD_ID=$(kubectl get pods -n=${GIT_SERVER_NS} -l app=test-git-server -o jsonpath='{.items[0].metadata.name}')

echo "Setting up remote git repo"
mkdir -p ${TEST_LOG_REPO}
kubectl -n="${GIT_SERVER_NS}" port-forward "${POD_ID}" "${FWD_SSH_PORT}:22" > "${TEST_LOG_REPO}/port-forward.log" &
# shellcheck disable=SC2191
REMOTE_GIT=(kubectl exec -n="${GIT_SERVER_NS}" "${POD_ID}" -- git)
"${REMOTE_GIT[@]}" init --bare --shared /git-server/repos/sot.git
"${REMOTE_GIT[@]}" \
  -C /git-server/repos/sot.git config receive.denyNonFastforwards false

echo "Setting up local git repo"
# git-sync wants the designated sync branch to exist, so we create a dummy
# commit so that the sync branch exists
export GIT_SSH_COMMAND="ssh -q -o StrictHostKeyChecking=no -i /opt/testing/e2e/id_rsa.nomos"
mkdir -p "${TEST_LOG_REPO}/repo"
cd "${TEST_LOG_REPO}/repo" || exit 1
git init
git remote add origin ssh://git@localhost:2222/git-server/repos/sot.git
git config user.name "Testing Nome"
git config user.email testing_nome@example.com
mkdir acme
touch acme/README.md
git add acme/README.md
git commit -a -m "initial commit"
git push origin master -f
echo "Finished setting up git"
