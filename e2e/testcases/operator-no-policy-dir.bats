#!/bin/bash

set -euo pipefail

load "../lib/git"
load "../lib/configmap"
load "../lib/setup"
load "../lib/wait"
load "../lib/nomos"

FILE_NAME="$(basename "${BATS_TEST_FILENAME}" '.bats')"

setup() {
  if "${RAW_NOMOS}"; then
    kubectl apply -f "${MANIFEST_DIR}/importer_policy-dir-acme.yaml"
    kubectl apply -f "${MANIFEST_DIR}/source_format_hierarchy.yaml"
    nomos::restart_pods
  fi

  setup::common
  setup::git::initialize
  setup::git::init acme
}

teardown() {
  setup::git::remove_all acme

  if "${RAW_NOMOS}"; then
    kubectl apply -f "${MANIFEST_DIR}/default-configmaps.yaml"
    nomos::restart_pods
  else
    kubectl apply -f "${MANIFEST_DIR}/operator-config-git.yaml"
  fi

  setup::common_teardown
}


@test "${FILE_NAME}: Absence of cluster, namespaces, systems directories at POLICY_DIR yields a missing repo error" {
  if "${RAW_NOMOS}"; then
    kubectl apply -f "${MANIFEST_DIR}/importer_no-policy-dir.yaml"
    nomos::restart_pods
  else
    kubectl apply -f "${MANIFEST_DIR}/operator-config-git-no-policy-dir.yaml"
  fi

  # Verify that the application of the operator config yields the correct error code
  wait::for -t 60 -o "KNV1017" -- kubectl get repo repo -o=jsonpath='{.status.import.errors[0].code}'
}

# This test case gradually adjusts root and acme directories' contents to move from:
# POLICY_DIR == "acme" to: POLICY_DIR undefined in the template.  Thus the gradual steps.
@test "${FILE_NAME}: Confirm that nomos syncs correctly with POLICY_DIR unset" {
  setup::git::add_contents_to_root acme

  if "${RAW_NOMOS}"; then
    kubectl apply -f "${MANIFEST_DIR}/importer_no-policy-dir.yaml"
    nomos::restart_pods
  else
    kubectl apply -f "${MANIFEST_DIR}/operator-config-git-no-policy-dir.yaml"
  fi

  setup::git::remove_folder acme

  wait::for -t 60 -- nomos::repo_synced
  wait::for -t 60 -o "" -- kubectl get repo repo -o=jsonpath='{.status.import.errors}'
}
