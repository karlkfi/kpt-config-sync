#!/bin/bash

set -euo pipefail

load "../lib/assert"
load "../lib/git"
load "../lib/setup"
load "../lib/wait"
load "../lib/resource"
load "../lib/nomos"

FILE_NAME="$(basename "${BATS_TEST_FILENAME}" '.bats')"

test_teardown() {
  repair_state
}

test_-24-7bFILE-5fNAME-7d-3a_Syncs_correctly_with_explicit_namespace_declarations() { bats_test_begin "${FILE_NAME}: Syncs correctly with explicit namespace declarations" 18; 
  kubectl apply -f "${MANIFEST_DIR}/source-format_unstructured.yaml"
  kubectl apply -f "${MANIFEST_DIR}/importer_repoless.yaml"
  nomos::restart_pods

  setup::git::initialize
  setup::git::init repoless

  wait::for -t 60 -- nomos::repo_synced

  resource::check clusterrole repoless-admin
  resource::check namespace backend
  resource::check_count -c 0 -r namespace -l "backend.tree.hnc.x-k8s.io=0"
  resource::check role pod-reader-backend -n backend
  resource::check namespace default
  resource::check_count -c 0 -r namespace -l "default.tree.hnc.x-k8s.io=0"
  resource::check_count -c 0 -r namespace -a "hnc.x-k8s.io/managedBy=configmanagement.gke.io"
  resource::check role pod-reader-default -n default
}

test_-24-7bFILE-5fNAME-7d-3a_Syncs_correctly_with_only_roles() { bats_test_begin "${FILE_NAME}: Syncs correctly with only roles" 38; 
  kubectl apply -f "${MANIFEST_DIR}/source-format_unstructured.yaml"
  kubectl apply -f "${MANIFEST_DIR}/importer_repoless-no-ns.yaml"
  nomos::restart_pods

  setup::git::initialize
  setup::git::init repoless-no-ns

  wait::for -t 60 -- nomos::repo_synced

  resource::check clusterrole repoless-admin
  resource::check namespace backend
  resource::check_count -c 0 -r namespace -l "backend.tree.hnc.x-k8s.io=0"
  resource::check role pod-reader-backend -n backend
  resource::check namespace default
  resource::check_count -c 0 -r namespace -l "default.tree.hnc.x-k8s.io=0"
  resource::check_count -c 0 -r namespace -a "hnc.x-k8s.io/managedBy=configmanagement.gke.io"
  resource::check role pod-reader-default -n default
}

test_-24-7bFILE-5fNAME-7d-3a_Syncs_correctly_with_no_policy_dir_set() { bats_test_begin "${FILE_NAME}: Syncs correctly with no policy dir set" 58; 
  kubectl apply -f "${MANIFEST_DIR}/source-format_unstructured.yaml"
  kubectl apply -f "${MANIFEST_DIR}/importer_no-policy-dir.yaml"
  nomos::restart_pods

  setup::git::initialize
  setup::git::init repoless

  wait::for -t 60 -- nomos::repo_synced

  resource::check clusterrole repoless-admin
  resource::check namespace backend
  resource::check_count -c 0 -r namespace -l "backend.tree.hnc.x-k8s.io=0"
  resource::check role pod-reader-backend -n backend
  resource::check namespace default
  resource::check_count -c 0 -r namespace -l "default.tree.hnc.x-k8s.io=0"
  resource::check_count -c 0 -r namespace -a "hnc.x-k8s.io/managedBy=configmanagement.gke.io"
  resource::check role pod-reader-default -n default
}

function repair_state() {
  kubectl apply -f "${MANIFEST_DIR}/default-configmaps.yaml"
  nomos::restart_pods
}

bats_test_function test_-24-7bFILE-5fNAME-7d-3a_Syncs_correctly_with_explicit_namespace_declarations
bats_test_function test_-24-7bFILE-5fNAME-7d-3a_Syncs_correctly_with_only_roles
bats_test_function test_-24-7bFILE-5fNAME-7d-3a_Syncs_correctly_with_no_policy_dir_set
