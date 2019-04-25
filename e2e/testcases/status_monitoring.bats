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

function token_exists() {
  local resource_type="${1}"
  local resource_name="${2}"
  local token_name=${3}

  ensure_error_free_repo

  run kubectl get "${resource_type}" "${resource_name}" \
    --output="jsonpath={${token_name}}"
  # shellcheck disable=SC2154
  if [[ "${output}" == "" ]]; then
    debug::log \
      "${resource_type} ${resource_type} ${token_name} is not filled in"
    debug::error "$(kubectl get "${resource_type}" "${resource_name}" -o yaml)"
  fi
}

@test "Version populated in ConfigManagement.status.configManagementVersion" {
  wait::for -t 60 -- token_exists \
    "configmanagement" "config-management" ".status.configManagementVersion"
}

@test "repo .status.import.token is populated" {
  wait::for -t 60 -- token_exists "repo" "repo" ".status.import.token"
}

@test "repo .status.source.token is populated" {
  wait::for -t 60 -- token_exists "repo" "repo" ".status.source.token"
}

@test "repo .status.sync.latestToken is populated" {
  wait::for -t 60 -- token_exists "repo" "repo" ".status.sync.latestToken"
}

function dummy_repo_update() {
  debug::log "Update something inconsequential in the repo"
  git::add "${YAML_DIR}/dir-namespace.yaml" acme/namespaces/dir/namespace.yaml
  git::commit -a -m "Update something inconsequential in the repo"
}

# token_updated returns true if the status token on the resource changes from
# its initially provided value.
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
    debug::log "initial token: ${token}; last read token: ${new_token}"
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

  # It may take quite a while from the time repo is error-free to the time
  # that is reflected in the repo status.  Wait for that to happen.
  wait::for -t 60 -- \
    token_exists "${resource_type}" "${resource_name}" "${token_name}"

  run kubectl get "${resource_type}" "${resource_name}" \
    --output="jsonpath={${token_name}}"
  # shellcheck disable=SC2154
  if [[ "${output}" == "" ]]; then
    debug::log \
      "${resource_type} ${resource_name} ${token_name} is not filled in."
    debug::error "$(kubectl get "${resource_type}" "${resource_name}" -o yaml)"
  fi

  local token="${output}"
  debug::log "stable token is: ${token}"
  dummy_repo_update

  debug::log "Check that the token was updated after repo update from: ${token}"
  wait::for -t 60 -- \
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

@test "repo .status.import.errors is populated on error" {
  ensure_error_free_repo
  wait::for -t 30 -f -- kubectl get namespace dir

  debug::log "Attempt to add two dirs with the same name"
  git::add "${YAML_DIR}/dir-namespace.yaml" acme/namespaces/foo/dir/namespace.yaml
  git::add "${YAML_DIR}/dir-namespace.yaml" acme/namespaces/bar/dir/namespace.yaml
  git::commit -a -m "Attempt to add two dirs with the same name"

  debug::log "Check that an error code is set on import status"
  wait::for -t 30 -o "KNV1002" -- \
    kubectl get repo repo --output='jsonpath={.status.import.errors[0].code}'
}

# A sync error happens if the managed objects are well-formed but some specific
# detail prevents an "apply" operation.  One instance is given below: we have
# a perfectly legit service in a managed namespace, except that its name is not
# valid.  Normally, `nomos vet` will notice this before the error gets into a
# cluster.
@test "repo .status.sync.errors is populated on error" {
  skip "Re-enable when b/131250908 is fixed"
  ensure_error_free_repo
  wait::for -t 30 -f -- kubectl get namespace dir

  debug::log \
    "Attempt to add a managed namespace with an invalid service in it"
  git::add "${YAML_DIR}/dir-namespace.yaml" acme/namespaces/dir/namespace.yaml
  git::add \
    "${YAML_DIR}/invalid/service-invalid-name.yaml" \
    acme/namespaces/dir/service.yaml
  git::commit -a \
    -m "Attempt to add a managed namespace with an invalid service in it"

  debug::log "Check that an error code is set on sync status"
  wait::for -t 30 -o "KNV2010" -- \
    kubectl get repo repo --output='jsonpath={.status.sync.errors[0].code}'
}

