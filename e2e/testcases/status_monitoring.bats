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

function examine_token() {
  local resource_type="${1}"
  local resource_name="${2}"
  local token_name=${3}
  debug::log "Ensure that the repo is error-free at the start of the test"
  git::commit -a -m "Commit the repo contents."
  wait::for -t 30 -o "true" -- kubectl get configmanagement config-management \
    --output='jsonpath={.status.healthy}'

  run kubectl get "${resource_type}" "${resource_name}" \
    --output="jsonpath={${token_name}}"
  # shellcheck disable=SC2154
  if [[ "${output}" == "" ]]; then
    debug::error \
      "${resource_type} ${resource_type} ${resource_name} is not filled in."
    debug::error "$(kubectl get "${resource_type}" "${resource_name}" -o yaml)"
  fi
}

@test "Version populated in ConfigManagement.status.configManagementVersion" {
  examine_token "configmanagement" "config-management" ".status.configManagementVersion"
}

@test "repo .status.import.token is populated" {
  examine_token "repo" "repo" ".status.import.token"
}

@test "repo .status.source.token is populated" {
  examine_token "repo" "repo" ".status.source.token"
}

@test "repo .status.sync.latestToken is populated" {
  examine_token "repo" "repo" ".status.sync.latestToken"
}
