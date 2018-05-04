#!/bin/bash

# End to end tests for Nomos.
# As a prerequisite for running this test, you must have
# - A 1.9 kubernetes cluster configured for nomos. See
# -- scripts/cluster/gce/kube-up.sh
# -- scripts/cluster/gce/configure-apserver-for-nomos.sh
# -- scripts/cluster/gce/configure-monitoring.sh
# - kubectl configured with context pointing to that cluster
# - Docker
# - gcloud with access to a project that has GCR

# To execute a subset of tests without setup, run as folows:
# > SKIP_INITIAL_SETUP=1 TEST_FUNCTIONS=testNomosResourceQuota e2e/e2e.sh

set -u

TEST_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
TEST_REPO_DIR=/tmp/nomos-test

# Run for every test
function setUp() {
  CWD=$(pwd)
  mkdir -p ${TEST_REPO_DIR}
  cd ${TEST_REPO_DIR}
  rm -rf repo
  git clone ssh://git@localhost:2222/git-server/repos/sot.git ${TEST_REPO_DIR}/repo
  cd ${TEST_REPO_DIR}/repo
  git rm -rf acme
  cp -r /opt/testing/sot/acme ./
  git add -A
  git commit -m "setUp commit"
  git push origin master
  cd $CWD
  # Wait for syncer to update objects following the policynode updates
  sleep 1
}

# assertContains <command> <substring>
# Will fail if the output of the command or its error message doesn't contain substring
function assertContains() {
  result=$(eval $1 2>&1)
  if [[ $result != *"$2"* ]]; then
    echo "FAIL: [$result] does not contain [$2]"
    exit 1
  fi
}

######################## TESTS ########################
function testSyncerNamespaces() {
  assertContains "kubectl get ns eng" "NotFound"
  assertContains "kubectl get ns backend" "Active"
  assertContains "kubectl get ns frontend" "Active"

  assertContains "kubectl get ns frontend -o yaml" 'nomos-managed: "true"'
  assertContains "kubectl get ns frontend -o yaml" "nomos-parent-ns: eng"
}

function testSyncerRoles() {
  assertContains "kubectl get roles -n new-prj" "acme-admin"
}

function testSyncerRoleBindings() {
  assertContains "kubectl get rolebindings -n backend bob-rolebinding -o yaml" "acme-admin"

  assertContains "kubectl get rolebindings -n backend -o yaml" "alice"
  assertContains "kubectl get rolebindings -n frontend -o yaml" "alice"
}

function testSyncerRoleBindingsChange() {
  kubectl apply -f test-syncer-change-rolebinding-backend.yaml > /dev/null
  sleep 1
  assertContains "kubectl get rolebindings -n backend bob-rolebinding" "NotFound"
  assertContains "kubectl get rolebindings -n backend robert-rolebinding -o yaml" "acme-admin"
}

function testSyncerQuota() {
  assertContains "kubectl get quota -n backend -o yaml" 'pods: "1"'
}

# Borked for now
function NOTtestAuhorizer() {
  assertContains "kubectl get pods -n backend --as bob@acme.com" "No resources"
  assertContains "kubectl get pods -n backend --as alice@acme.com" "No resources"

  assertContains "kubectl get pods -n frontend --as bob@acme.com" "pods is forbidden"
}

# Helper for quota tests
function cleanTestConfigMaps() {
  kubectl delete configmaps -n new-prj --all > /dev/null
  kubectl delete configmaps -n newer-prj --all > /dev/null
  sleep 1
}

function testQuotaAdmission() {
  cleanTestConfigMaps
  waitForSuccess "kubectl get ns new-prj"
  assertContains "kubectl create configmap map1 -n new-prj" "created"
  assertContains "kubectl create configmap map2 -n newer-prj" "created"
  assertContains "kubectl create configmap map3 -n new-prj" "exceeded quota in policyspace rnd"
  cleanTestConfigMaps
}

function waitForSuccess() {
  local command="${1:-}"
  local timeout=${2:-10}
  local or_die=${3:-true}
  waitFor "${command}" "${timeout}" "${or_die}" true
}

function waitForFailure() {
  local command="${1:-}"
  local timeout=${2:-10}
  local or_die=${3:-true}
  waitFor "${command}" "${timeout}" "${or_die}" false
}

function waitFor() {
  local command="${1:-}"
  local timeout=${2:-10}
  local or_die=${3:-true}
  local expect=${4:-true}

  echo -n "Waiting for ${command} to exit ${expect}"
  for i in $(seq 1 ${timeout}); do
    if ${command} &> /dev/null; then
      if ${expect}; then
        echo
        return 0
      fi
    else
      if ! ${expect}; then
        echo
        return 0
      fi
    fi
    echo -n "."
    sleep 0.5
  done
  echo
  echo "Command '${command}' failed after ${timeout} seconds"
  if ${or_die}; then
    exit 1
  fi
}

TEST_FUNCTIONS=${TEST_FUNCTIONS:-$(declare -F)}

######################## MAIN #########################
function setUpEnv() {
  echo "****************** Setting up environment ******************"
  readonly kubeconfig_output="/opt/installer/kubeconfig/config"
  # We need to fix up the kubeconfig paths because these may not match between
  # the container and the host.
  # /somepath/gcloud becomes /use/local/gcloud/google-cloud/sdk/bin/gcloud.
  # Then, make it read-writable to the owner only.
  cat /home/user/.kube/config | \
    sed -e "s+cmd-path: [^ ]*gcloud+cmd-path: /usr/local/gcloud/google-cloud-sdk/bin/gcloud+g" \
    > "${kubeconfig_output}"
  chmod 600 ${kubeconfig_output}

  suggested_user="$(gcloud config get-value account)"

  /opt/testing/init-git-server.sh

  ./installer \
    --config="${TEST_DIR}/install-config.yaml" \
    --log_dir=/tmp \
    --suggested_user="${suggested_user}" \
    --use_current_context=true \
    --vmodule=main=10,configgen=10,kubectl=10,installer=10,exec=10
  echo "****************** Environment is ready ******************"
}

function main() {
  if [[ ! "kubectl get ns > /dev/null" ]]; then
    echo "Kubectl/Cluster misconfigured"
    exit 1
  fi
  GIT_SSH_COMMAND="ssh -q -o StrictHostKeyChecking=no -i /opt/testing/id_rsa.nomos"; export GIT_SSH_COMMAND

  cd ${TEST_DIR}
  echo "****************** Starting tests ******************"
  # for each function starting with "test"
  for test in $TEST_FUNCTIONS; do
    if [[ $test == "test"* ]]; then
      setUp
      echo -n "$test: "
      start=`date +%s`
      $test
      end=`date +%s`
      duration=$((end-start))
      echo "PASS (${duration}s)"
    fi
  done
}

function cleanUp() {
  echo "****************** Cleaning up environment ******************"
  kubectl delete ValidatingWebhookConfiguration policy-nodes.nomos.dev --ignore-not-found
  kubectl delete ValidatingWebhookConfiguration resource-quota.nomos.dev --ignore-not-found
  kubectl delete policynodes --all || true
  kubectl delete clusterpolicy --all || true
  kubectl delete --ignore-not-found ns nomos-system
  ! pkill -f "kubectl -n=nomos-system port-forward.*2222:22"

  echo "Deleting namespaces nomos-system, this may take a minute"
  while kubectl get ns nomos-system > /dev/null 2>&1
  do
    sleep 3
    echo -n "."
  done
}

cleanUp
setUpEnv
main
cleanUp
