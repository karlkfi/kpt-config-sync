#!/bin/bash

set -euo pipefail

load "../lib/assert"
load "../lib/git"
load "../lib/setup"
load "../lib/wait"
load "../lib/resource"

readonly YAML_DIR="${BATS_TEST_DIRNAME}/../testdata"

FILE_NAME="$(basename "${BATS_TEST_FILENAME}" '.bats')"

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
  setup::git::remove_all acme

  # Force delete repo object.
  kubectl delete repo repo --ignore-not-found

  setup::common_teardown
}

function ensure_error_free_repo () {
  debug::log "Ensure that the repo is error-free at the start of the test"
  git::commit -a -m "Commit the repo contents."
  wait::for -t 30 -o "true" -- kubectl get configmanagement config-management \
    --output='jsonpath={.status.healthy}'
}

@test "${FILE_NAME}: Invalid policydir gets an error message in status.source.errors" {

  ensure_error_free_repo
  debug::log "Setting policyDir to acme"
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
  wait::for -t 90 -c "some-nonexistent-policydir" -- \
    kubectl get repo repo -o=yaml \
      --output='jsonpath={.status.source.errors[0].errorMessage}'

  debug::log "Fix the policydir in the repo"
  kubectl patch configmanagement config-management \
    --type=merge \
    --patch '{"spec":{"git":{"policyDir":"acme"}}}'

  debug::log "Expect repo to recover from the error in source message"
  wait::for -t 90 -o "" -- kubectl get repo repo \
    --output='jsonpath={.status.source.errors[0].errorMessage}'
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

@test "${FILE_NAME}: Version populated in ConfigManagement.status.configManagementVersion" {
  wait::for -t 60 -- token_exists \
    "configmanagement" "config-management" ".status.configManagementVersion"
}

@test "${FILE_NAME}: repo .status.import.token is populated" {
  skip # TODO(b/138198997): Make non-flaky or remove.
  wait::for -t 60 -- token_exists "repo" "repo" ".status.import.token"
}

@test "${FILE_NAME}: repo .status.source.token is populated" {
  skip # TODO(b/138198997): Make non-flaky or remove.
  wait::for -t 60 -- token_exists "repo" "repo" ".status.source.token"
}

@test "${FILE_NAME}: repo .status.sync.latestToken is populated" {
  skip # TODO(b/138198997): Make non-flaky or remove.
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
    return 1
  fi
  return 0
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

function get_last_update() {
  local resource_type="${1}"
  local resource_name="${2}"
  local component_name="${3}"

  # Ensure repo exists before checking it.
  wait::for -t 30 -- kubectl get repo repo
  run kubectl get "${resource_type}" "${resource_name}" \
      --output="jsonpath={.status.${component_name}.lastUpdate}"
}

function timestamp_updated() {
  local resource_type="${1}"
  local resource_name="${2}"
  local component_name="${3}"
  local last_update="${4}"

  local new_update
  new_update="$(kubectl get "${resource_type}" "${resource_name}" \
      --output="jsonpath={.status.${component_name}.lastUpdate}")"

  if [[ "${new_update}" == "${last_update}" ]]; then
    # Fail, timestamp not updated
    debug::log "initial timestamp: ${last_update}; last read timestamp: ${new_update}"
    return 1
  fi
  return 0
}

@test "${FILE_NAME}: repo .status.import.token is updated" {
  skip # TODO(b/138198997): Make non-flaky or remove.
  ensure_token_updated "repo" "repo" ".status.import.token"
}

@test "${FILE_NAME}: repo .status.source.token is updated" {
  skip # TODO(b/138198997): Make non-flaky or remove.
  ensure_token_updated "repo" "repo" ".status.source.token"
}

@test "${FILE_NAME}: repo .status.sync.latestToken is updated" {
  skip # TODO(b/138198997): Make non-flaky or remove.
  ensure_token_updated "repo" "repo" ".status.sync.latestToken"
}

@test "${FILE_NAME}: repo .status.import.errors is populated on error" {
  skip # TODO(b/138198997): Make non-flaky or remove.
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
@test "${FILE_NAME}: repo .status.sync.errors is populated on error" {
  skip # TODO(b/138198997): Make non-flaky or remove.
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
    kubectl get repo repo \
      --output='jsonpath={.status.sync.inProgress[0].errors[0].code}'
}

@test "${FILE_NAME}: repo .status.sync.token is not populated on initial invalid repo" {
  skip # TODO(b/138198997): Make non-flaky or remove.
  get_last_update "repo" "repo" "sync"
  local last_update="${output}"

  debug::log "Ensure errorful repo by adding two dirs with the same name"
  git::add "${YAML_DIR}/dir-namespace.yaml" acme/namespaces/foo/dir/namespace.yaml
  git::add "${YAML_DIR}/dir-namespace.yaml" acme/namespaces/bar/dir/namespace.yaml
  git::commit -a -m "Attempt to add two dirs with the same name"

  debug::log "Check that an error code is set on import status"
  wait::for -t 30 -o "KNV1002" -- \
    kubectl get repo repo --output='jsonpath={.status.import.errors[0].code}'

  debug::log "Check that sync timestamp is updated"
  wait::for -t 60 -- \
      timestamp_updated "repo" "repo" "sync" "${last_update}"

  debug::log "Check that sync token is not updated"
  run kubectl get repo repo --output="jsonpath={.status.sync.latestToken}"
  # shellcheck disable=SC2154
  if [[ "${output}" != "" ]]; then
    debug::log "repo repo .status.sync.latestToken is filled in and should not be."
    debug::error "$(kubectl get repo repo -o yaml)"
  fi
}
