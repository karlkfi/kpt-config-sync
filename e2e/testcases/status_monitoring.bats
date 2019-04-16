#!/bin/bash

set -euo pipefail

load "../lib/assert"
load "../lib/git"
load "../lib/setup"
load "../lib/wait"
load "../lib/resource"

setup() {
  setup::common
  setup::git::initialize

  local TEST_REPO_DIR=${BATS_TMPDIR}
  cd "${TEST_REPO_DIR}/repo"

  mkdir -p acme/system
  cp -r /opt/testing/e2e/examples/acme/system acme
  git add -A
}

teardown() {
  setup::common_teardown
}

@test "Invalid policydir gets an error message in status.source.errors" {
  debug::log "Ensure that the repo is error-free at the start of the test"
  git::commit -a -m "Commit the repo contents."
  kubectl patch configmanagement config-management \
    --type=merge \
    --patch '{"spec":{"git":{"policyDir":"acme"}}}'
  wait::for -t 10 -o "" -- kubectl get repo repo \
    --output='jsonpath={.status.import.errors[0].errorMessage}'
  wait::for -t 10 -o "" -- kubectl get repo repo \
    --output='jsonpath={.status.source.errors[0].errorMessage}'

  debug::log "Break the policydir in the repo"
  kubectl patch configmanagement config-management \
    --type=merge \
    --patch '{"spec":{"git":{"policyDir":"some-nonexistent-policydir"}}}'

  # Increased timeout from initial 30 to 60 for flakiness.  git-importer
  # gets restarted on each object change.
  debug::log "Expect an error to be present in status.source.errors"
  wait::for -t 60 -c "some-nonexistent-policydir" -- \
    kubectl get repo repo -o=yaml \
      --output='jsonpath={.status.source.errors[0].errorMessage}'

  debug::log "Fix the policydir in the repo"
  kubectl patch configmanagement config-management \
    --type=merge \
    --patch '{"spec":{"git":{"policyDir":"acme"}}}'

  debug::log "Expect repo to recover from the error in source message"
  wait::for -t 60 -o "" -- kubectl get repo repo \
    --output='jsonpath={.status.source.errors[0].errorMessage}'
}

@test "Version populated in ConfigManagement.status.configManagementVersion" {
  debug::log "Ensure that the repo is error-free at the start of the test"
  git::commit -a -m "Commit the repo contents."
  wait::for -t 30 -o "true" -- kubectl get configmanagement config-management \
    --output='jsonpath={.status.healthy}'

  run kubectl get configmanagement config-management \
    --output='jsonpath={.status.configManagementVersion}'
  # shellcheck disable=SC2154
  if [[ "${output}" == "" ]]; then
    debug::error ".status.ConfigManagementVersion is not filled in."
    debug::error "$(kubectl get configmanagement config-management -o yaml)"
  fi 
}
