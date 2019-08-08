#!/bin/bash

# This script runs Syllogi specifc end to end tests that should be run
# against a Syllogi-created cluster. The script is the entry point of
# the gcr.io/syllogi-ci/e2e-nomos-syllogi:latest image which is run
# by the Syllogi suite of conformance tests.

set -euo pipefail

readonly TEST_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# shellcheck source=e2e/lib/wait.bash
source "$TEST_DIR/lib/wait.bash"
# shellcheck source=e2e/lib/install.bash
source "$TEST_DIR/lib/install.bash"
# shellcheck source=e2e/lib/resource.bash
source "$TEST_DIR/lib/resource.bash"

function install() {
  # Setup git
  install::create_keypair
  /opt/testing/e2e/init-git-server.sh
  export GIT_SSH_COMMAND="ssh -q -o StrictHostKeyChecking=no -i $TEST_DIR/id_rsa.nomos"
  kubectl create secret generic git-creds \
    -n=config-management-system --from-file=ssh="$TEST_DIR/id_rsa.nomos" || true

  # Install Nomos
  kubectl apply -f "${TEST_DIR}/operator-config-syllogi.yaml"

  echo "Waiting for Nomos to come up"
  wait::for -s -t 180 -- install::nomos_running
  local image
  image="$(kubectl get pods -n config-management-system -l app=syncer -ojsonpath='{.items[0].spec.containers[0].image}')"
  echo "Nomos $image up and running"
}

function cleanup() {
  echo "printing diagonstics"
  echo "+ operator logs"
  (kubectl -n kube-system logs -l k8s-app=config-management-operator --tail=100) || true
  echo "+ importer pod"
  (kubectl -n config-management-system describe pod git-importer) || true
  echo "+ importer logs"
  (kubectl -n config-management-system logs -l app=git-importer -c importer --tail=100) || true
  echo "cleaning up"
  kubectl delete configmanagement config-management --ignore-not-found
  wait::for -s -t 180 -- install::nomos_uninstalled
  echo "Nomos uninstalled"

  kubectl delete --ignore-not-found ns -l "configmanagement.gke.io/testdata=true"
  resource::delete -r ns -a configmanagement.gke.io/managed=enabled

  echo "killing kubectl port forward..."
  pkill -f "kubectl -n=config-management-system-test port-forward.*2222:22" || true
  echo "taking down config-management-system-test namespace"
  kubectl delete --ignore-not-found ns config-management-system-test
  wait::for -f -t 100 -- kubectl get ns config-management-system-test
  kubectl delete --ignore-not-found secrets git-creds -n config-management-system

  echo "Nomos test resources uninstalled"
}

# Main

cleanup

install

# Always clean up on exit
trap cleanup EXIT

# Trapping EXIT without trapping SIGTERM/SIGINT causes the docker container
# running the tests to fail to exit.
# More detail from our experience report here:
# https://stackoverflow.com/questions/54699272/docker-hangs-on-sigint-when-script-traps-exit
# Note that ":" is the bashism for no-op; it evals to true.
trap : SIGTERM SIGINT

exitcode=0
echo "Starting Nomos tests"
result_file="${REPORT_DIR}/results.bats"

# Compute test times
export TIMING=true

# We only run the foo-corp test to have a minimal sanity test that the version that
# comes with syllogi works. Detailed functionality tests are left to our own tests.
run_bats_tests=("${TEST_DIR}/bats/bin/bats" "--tap" "${TEST_DIR}/testcases/foo_corp.bats")
if ! "${run_bats_tests[@]}" | tee "${result_file}"; then
  echo "Failed Nomos tests"
  exitcode=1
fi

echo "Converting test results from TAP format to jUnit"
/opt/tap2junit -reorder_duration -test_name "[Nomos Addon]" \
  < "${result_file}" \
  > "${REPORT_DIR}/junit_nomos.xml"

echo "Results at ${REPORT_DIR}/junit_nomos.xml"

exit ${exitcode}
