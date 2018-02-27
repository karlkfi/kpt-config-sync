#!/bin/bash

# End to end tests for Stolos.
# As a prerequisite for running this test, you must have
# - A 1.9 kubernetes cluster configured for stolos. See
# -- scripts/cluster/gce/kube-up.sh
# -- scripts/cluster/gce/configure-apserver-for-stolos.sh
# -- scripts/cluster/gce/configure-monitoring.sh
# - kubectl configured with context pointing to that cluster
# - Docker
# - gcloud with access to a project that has GCR

# To execute a subset of tests without setup, run as folows:
# > SKIP_INITIAL_SETUP=1 TEST_FUNCTIONS=testStolosResourceQuota e2e/e2e.sh

set -u

# Should be $STOLOS/e2e
TESTDIR="$( cd "$(dirname "$0")" ; pwd -P )"
# Should be $STOLOS/
MAKEDIR=$TESTDIR/..
# Path to demo acme yaml
ACME=$MAKEDIR/examples/acme/policynodes/acme.yaml

# Run for every test
function setUp() {
  kubectl apply -f ${ACME} > /dev/null
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

  assertContains "kubectl get ns frontend -o yaml" 'stolos-managed: "true"'
  assertContains "kubectl get ns frontend -o yaml" "stolos-parent-ns: eng"
}

function testSyncerRoles() {
  assertContains "kubectl get roles -n backend" "acme-admin"
  assertContains "kubectl get roles -n frontend" "acme-admin"
}

function testSyncerRoleBindings() {
  assertContains "kubectl get rolebindings -n backend bob-rolebinding -o yaml" "acme-admin"

  assertContains "kubectl get rolebindings -n backend -o yaml" "alice"
  assertContains "kubectl get rolebindings -n frontend -o yaml" "alice"
}

function testSyncerRoleBindingsChange() {
  kubectl apply -f ${TESTDIR}/test-syncer-change-rolebinding-backend.yaml > /dev/null
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
function main() {
  if [[ ! "kubectl get ns > /dev/null" ]]; then
    echo "Kubectl/Cluster misconfigured"
    exit 1
  fi

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

main
