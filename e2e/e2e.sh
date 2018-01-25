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

# To execute the test without one time setup
# > SKIP_INITIAL_SETUP=1 e2e/e2e.sh

# To execute a subset of tests without setup, run as folows:
# > SKIP_INITIAL_SETUP=1 TEST_FUNCTIONS=testStolosResourceQuota e2e/e2e.sh

# To execute the tests with final clean up
# > SKIP_FINAL_CLEANUP=0 e2e/e2e.sh

set -u

# Should be $STOLOS/e2e
TESTDIR="$( cd "$(dirname "$0")" ; pwd -P )"
# Should be $STOLOS/
MAKEDIR=$TESTDIR/..
# Path to demo acme yaml
ACME=$MAKEDIR/examples/acme/acme.yaml

function cleanUp() {
  echo "****************** Cleaning up environment ******************"
  kubectl delete policynodes --all
  kubectl delete ValidatingWebhookConfiguration stolos-resource-quota --ignore-not-found
  kubectl delete ns stolos-system

  echo "Deleting namespace stolos-system, this may take a minute"
  while kubectl get ns stolos-system > /dev/null 2>&1
  do
    sleep 3
    echo -n "."
  done
  echo
}

# Run once
function initialSetUp() {
  cleanUp
  echo "****************** Setting up environment ******************"
  cd ${MAKEDIR}

  make deploy-common-objects

  if ! kubectl get ns stolos-system > /dev/null; then
    echo "Failed setting up Stolos on cluster"
    exit 1
  fi

  make deploy-all
  kubectl apply -f ${ACME}
  sleep 3
}

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
    exit
  fi
}

######################## TESTS ########################
function testSyncerNamespaces() {
  assertContains "kubectl get ns eng" "Active"
  assertContains "kubectl get ns acme" "Active"
  assertContains "kubectl get ns backend" "Active"
  assertContains "kubectl get ns frontend" "Active"

  assertContains "kubectl get ns frontend -o yaml" 'stolos-managed: "true"'
  assertContains "kubectl get ns frontend -o yaml" "stolos-parent-ns: eng"
  assertContains "kubectl get ns acme -o yaml" 'stolos-parent-ns: ""'
}

function testSyncerNamespaceDelete() {
  kubectl delete pn new-prj > /dev/null
  sleep 1
  # Takes a while for the namespace to get deleted, so we can just check for terminating
  assertContains "kubectl get ns new-prj" "Terminating"
}

function testSyncerRoles() {
  # Not Flattening
  assertContains "kubectl get roles -n acme" "admin"

  # Flattening
  assertContains "kubectl get roles -n backend" "admin"
  assertContains "kubectl get roles -n frontend" "admin"
}

function testSyncerRoleBindings() {
  assertContains "kubectl get rolebindings -n backend bob-rolebinding -o yaml" "admin"

  # Not Flattening
  assertContains "kubectl get rolebindings -n eng -o yaml" "alice"

  # Flattening
  assertContains "kubectl get rolebindings -n backend -o yaml" "alice"
  assertContains "kubectl get rolebindings -n frontend -o yaml" "alice"
}

function testSyncerRoleBindingsChange() {
  kubectl apply -f ${TESTDIR}/test-syncer-change-rolebinding-backend.yaml > /dev/null
  sleep 1
  assertContains "kubectl get rolebindings -n backend bob-rolebinding" "NotFound"
  assertContains "kubectl get rolebindings -n backend robert-rolebinding -o yaml" "admin"
}

function testSyncerQuota() {
  assertContains "kubectl get quota -n backend -o yaml" 'pods: "1"'
}

# Borked for now
function NOTtestAuhorizer() {
  assertContains "kubectl get pods -n backend --as bob@google.com" "No resources"
  assertContains "kubectl get pods -n backend --as alice@google.com" "No resources"

  assertContains "kubectl get pods -n frontend --as bob@google.com" "pods is forbidden"
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
  assertContains "kubectl create configmap map3 -n new-prj" "exceeded quota in namespace rnd"
  cleanTestConfigMaps
}

function testStolosResourceQuota() {
  cleanTestConfigMaps
  assertContains "kubectl get srq -n rnd -o yaml" 'configmaps: "2"' # limit
  assertContains "kubectl get srq -n rnd -o yaml" 'configmaps: "0"' # used
  assertContains "kubectl create configmap map1 -n new-prj" "created"
  sleep 1
  assertContains "kubectl get srq -n rnd  -o yaml" 'configmaps: "1"' # new used
}

SKIP_INITIAL_SETUP=${SKIP_INITIAL_SETUP:-0}
SKIP_FINAL_CLEANUP=${SKIP_FINAL_CLEANUP:-1}
TEST_FUNCTIONS=${TEST_FUNCTIONS:-$(declare -F)}

######################## MAIN #########################
function main() {
  if [[ ! "kubectl get ns > /dev/null" ]]; then
    echo "Kubectl/Cluster misconfigured"
    exit 1
  fi

  if [[ "$SKIP_INITIAL_SETUP" != 1 ]]; then
    initialSetUp
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

  if [[ "$SKIP_FINAL_CLEANUP" != 1 ]]; then
    cleanUp
  fi
}

main