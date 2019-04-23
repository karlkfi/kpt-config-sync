#!/bin/bash

set -euo pipefail

load "../lib/assert"
load "../lib/git"
load "../lib/setup"
load "../lib/wait"
load "../lib/resource"

readonly YAML_DIR="${BATS_TEST_DIRNAME}/../testdata"

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

function ensure_error_free_repo () {
  debug::log "Ensure that the repo is error-free at the start of the test"
  git::commit -a -m "Commit the repo contents."
  wait::for -t 30 -o "true" -- kubectl get configmanagement config-management \
    --output='jsonpath={.status.healthy}'
}

function examine_token() {
  local resource_type="${1}"
  local resource_name="${2}"
  local token_name=${3}

  ensure_error_free_repo

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
  examine_token \
    "configmanagement" "config-management" ".status.configManagementVersion"
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

function dummy_repo_update() {
  debug::log "Update something inconsequential in the repo"
  git::add "${YAML_DIR}/dir-namespace.yaml" acme/namespaces/dir/namespace.yaml
  git commit -a -m "Updating something inconsequential in the repo"
}

# token_updated returns true if the status token on the resource changes from
# its initial value.
#
# Example:
#   token_updated "repo" "repo" ".status.import.token" "deadbeef"
function token_updated() {
  local resource_type="${1}"
  local resource_name="${2}"
  local token_name="${3}"
  local token="${4}"

  local new_token
  new_token="$(kubectl get \
    "${resource_type}" "${resource_name}" --output="jsonpath={${token_name}}")"
  if [[ "${new_token}" == "${token}" ]]; then
    # Fail, token not updated
    return 0
  fi
  return 1
}

# Checks whether the value of the token with the passed-in name changes when
# repo contents change.
function ensure_token_updated() {
  local resource_type="${1}"
  local resource_name="${2}"
  local token_name="${3}"

  ensure_error_free_repo

  run kubectl get "${resource_type}" "${resource_name}" \
    --output="jsonpath={${token_name}}"
  # shellcheck disable=SC2154
  if [[ "${output}" == "" ]]; then
    debug::error \
      "${resource_type} ${resource_type} ${resource_name} is not filled in."
    debug::error "$(kubectl get "${resource_type}" "${resource_name}" -o yaml)"
  fi

  local token="${output}"
  dummy_repo_update

  debug::log "Check that the token was updated after repo update from: ${token}"
  wait::for -t 30 -- \
    token_updated "${resource_type}" "${resource_name}" \
      "${token_name}" "${token}"
}

@test "repo .status.import.token is updated" {
  ensure_token_updated "repo" "repo" ".status.import.token"
}

@test "repo .status.source.token is updated" {
  ensure_token_updated "repo" "repo" ".status.source.token"
}

@test "repo .status.sync.latestToken is updated" {
  ensure_token_updated "repo" "repo" ".status.sync.latestToken"
}
