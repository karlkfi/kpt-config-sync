#!/bin/bash

set -euo pipefail

load "../lib/assert"
load "../lib/env"
load "../lib/git"
load "../lib/setup"
load "../lib/wait"
load "../lib/resource"
load "../lib/nomos"

readonly YAML_DIR="${BATS_TEST_DIRNAME}/../testdata"

FILE_NAME="$(basename "${BATS_TEST_FILENAME}" '.bats')"

test_setup() {
  setup::git::initialize
  setup::git::commit_minimal_repo_contents --skip_commit
}

test_teardown() {
  setup::git::remove_all acme

  # Force delete repo object.
  if ! env::csmr; then
    kubectl delete repo repo --ignore-not-found
  fi
}

function ensure_error_free_repo () {
  debug::log "Ensure that the repo is error-free at the start of the test"
  git::commit -a -m "Commit the repo contents."
  wait::for nomos::repo_synced
  nomos::import_error_code ""
}

@test "${FILE_NAME}: Invalid policydir gets an error message in status.source.errors" {
  ensure_error_free_repo

  debug::log "Setting policyDir to acme"

  kubectl apply -f "${MANIFEST_DIR}/importer_acme.yaml"
  nomos::restart_pods
  nomos::import_error_code ""
  nomos::source_error_code ""

  debug::log "Break the policydir in the repo"
  if env::csmr; then
    kubectl patch -n config-management-system rootsyncs.configsync.gke.io root-sync \
      --type merge --patch '{"spec":{"git":{"dir":"some-nonexistent-policydir"}}}'
  else
    kubectl apply -f "${MANIFEST_DIR}/importer_some-nonexistent-policydir.yaml"
    nomos::restart_pods
  fi

  # Increased timeout from initial 30 to 60 for flakiness.  git-importer
  # gets restarted on each object change.
  debug::log "Expect an error to be present in status.source.errors"
  if env::csmr; then
    nomos::source_error_code "2001"
  else
    nomos::source_error_code "2004"
  fi

  debug::log "Fix the policydir in the repo"
  if env::csmr; then
    kubectl patch -n config-management-system rootsyncs.configsync.gke.io root-sync \
      --type merge --patch '{"spec":{"git":{"dir":"acme"}}}'
  else
    kubectl apply -f "${MANIFEST_DIR}/importer_acme.yaml"
    nomos::restart_pods
  fi

  debug::log "Expect repo to recover from the error in source message"
  nomos::source_error_code ""
}

function namespace_count_matches () {
  local expected=$1
  # the safety namespace (see lib/setup.bash) may be terminating, ignore it
  local actual=""
  actual=$(kubectl get ns --no-headers | grep -c -v safety)
  [ "${expected}" = "${actual}" ]
}

function check_acme_ns() {
  local -a ACME_NS
  # shellcheck disable=SC2038
  mapfile -t ACME_NS < <(find acme -name namespace.yaml | xargs dirname | sed 's|.*/||')
  for ns in "${ACME_NS[@]}"; do
    kubectl get ns "$ns"
  done
}

@test "${FILE_NAME}: Deleting all namespaces gets an error message in status.source.errors" {
  cd "${TEST_REPO}"
  mkdir -p acme/namespaces
  cp -r "${NOMOS_DIR}/examples/acme/namespaces" acme
  git add -A
  ensure_error_free_repo
  check_acme_ns

  debug::log "Delete all the namespaces (oops!)"
  cd "${TEST_REPO}"
  rm -rf "${TEST_REPO:?}/acme/namespaces"
  git add -A
  git::commit -a -m "wiping all namespaces from acme"

  debug::log "Expect an error to be present in status.source.errors"
  nomos::source_error_code "2006"

  debug::log "Before restore: ensure we still have the acme namespaces we expect"
  check_acme_ns

  debug::log "Restore the namespaces"
  cd "${TEST_REPO}"
  cp -r "${NOMOS_DIR}/examples/acme/namespaces" acme
  git add -A
  git::commit -a -m "restoring the namespaces"

  debug::log "Expect repo to recover from the error in source message"
  nomos::source_error_code ""

  debug::log "After restore: ensure we still have the acme namespaces we expect"
  check_acme_ns

  setup::git::remove_all acme
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

# TODO(b/158796412): Move this test to the operator
@test "${FILE_NAME}: Version populated in ConfigManagement.status.configManagementVersion" {
  skip "not testing unless test includes operator"

  wait::for -t 60 -- token_exists \
    "configmanagement" "config-management" ".status.configManagementVersion"
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
