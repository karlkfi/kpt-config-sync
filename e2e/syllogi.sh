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

function install() {
  # Setup git
  install::create_keypair
  /opt/testing/e2e/init-git-server.sh
  export GIT_SSH_COMMAND="ssh -q -o StrictHostKeyChecking=no -i /opt/testing/e2e/id_rsa.nomos"
  kubectl create secret generic git-creds -n=nomos-system --from-file=ssh="$HOME/.ssh/id_rsa.nomos" || true

  # Install Nomos
  kubectl apply -f "${TEST_DIR}/operator-config-syllogi.yaml"

  echo "Waiting for Nomos to come up"
  wait::for -s -t 180 -- install::nomos_running
  local image
  image="$(kubectl get pods -n nomos-system -l app=syncer -ojsonpath='{.items[0].spec.containers[0].image}')"
  echo "Nomos $image up and running"
}

function cleanup() {
  echo "cleaning up"
  kubectl delete nomos nomos --ignore-not-found -n nomos-system
  wait::for -s -t 180 -- install::nomos_uninstalled
  echo "Nomos uninstalled"

  kubectl delete --ignore-not-found ns -l "nomos.dev/testdata=true"
  kubectl delete --ignore-not-found ns -l "nomos.dev/managed=enabled"

  echo "killing kubectl port forward..."
  pkill -f "kubectl -n=nomos-system-test port-forward.*2222:22" || true
  echo "taking down nomos-system-test namespace"
  kubectl delete --ignore-not-found ns nomos-system-test
  kubectl delete --ignore-not-found secrets git-creds -n nomos-system

  echo "Nomos test resources uninstalled"
}

# Main

cleanup

install

trap cleanup EXIT
exitcode=0
echo "Starting Nomos tests"
result_file="${REPORT_DIR}/results.bats"
# We only run the foo-corp test to have a minimal sanity test that the version that
# comes with syllogi works. Detailed functionality tests are left to our own tests.
run_bats_tests=("${TEST_DIR}/bats/bin/bats" "--tap" "${TEST_DIR}/testcases/foo_corp.bats")
if ! "${run_bats_tests[@]}" | tee "${result_file}"; then
  echo "Failed Nomos tests"
  exitcode=1
fi

echo "Converting test results from TAP format to jUnit"
# The tap2junit util cleans up the test name and doesn't allow []s, but all syllogi tests
# start with "[test name]", so doing this silly thing here.
tap2junit --name "ZZPREFIX" "${result_file}"
sed "s/ZZPREFIX/[Nomos Addon] /" "${result_file}".xml > "${REPORT_DIR}/junit_nomos.xml"
echo "Results at ${REPORT_DIR}/junit_nomos.xml"

exit ${exitcode}