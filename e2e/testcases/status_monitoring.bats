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

