#!/bin/bash

set -euo pipefail

load "../lib/git"
load "../lib/configmap"
load "../lib/setup"
load "../lib/wait"
load "../lib/nomos"

FILE_NAME="$(basename "${BATS_TEST_FILENAME}" '.bats')"

test_setup() {
  kubectl apply -f "${MANIFEST_DIR}/importer_acme.yaml"
  kubectl apply -f "${MANIFEST_DIR}/source-format_hierarchy.yaml"
  nomos::restart_pods

  setup::git::initialize
  setup::git::init acme
}

test_teardown() {
  setup::git::remove_all acme

  kubectl apply -f "${MANIFEST_DIR}/default-configmaps.yaml"
  nomos::restart_pods

}


test_-24-7bFILE-5fNAME-7d-3a_Absence_of_cluster-2c_namespaces-2c_systems_directories_at_POLICY-5fDIR_yields_a_missing_repo_error() { bats_test_begin "${FILE_NAME}: Absence of cluster, namespaces, systems directories at POLICY_DIR yields a missing repo error" 31; 
  kubectl apply -f "${MANIFEST_DIR}/importer_no-policy-dir.yaml"
  nomos::restart_pods

  # Verify that the application of the operator config yields the correct error code
  wait::for -t 60 -o "KNV1017" -- kubectl get repo repo -o=jsonpath='{.status.import.errors[0].code}'
}

# This test case gradually adjusts root and acme directories' contents to move from:
# POLICY_DIR == "acme" to: POLICY_DIR undefined in the template.  Thus the gradual steps.
test_-24-7bFILE-5fNAME-7d-3a_Confirm_that_nomos_syncs_correctly_with_POLICY-5fDIR_unset() { bats_test_begin "${FILE_NAME}: Confirm that nomos syncs correctly with POLICY_DIR unset" 41; 
  setup::git::add_contents_to_root acme

  kubectl apply -f "${MANIFEST_DIR}/importer_no-policy-dir.yaml"
  nomos::restart_pods

  setup::git::remove_folder acme

  wait::for -t 60 -- nomos::repo_synced
  wait::for -t 60 -o "" -- kubectl get repo repo -o=jsonpath='{.status.import.errors}'
}

bats_test_function test_-24-7bFILE-5fNAME-7d-3a_Absence_of_cluster-2c_namespaces-2c_systems_directories_at_POLICY-5fDIR_yields_a_missing_repo_error
bats_test_function test_-24-7bFILE-5fNAME-7d-3a_Confirm_that_nomos_syncs_correctly_with_POLICY-5fDIR_unset
