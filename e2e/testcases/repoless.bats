#!/bin/bash

set -euo pipefail

load "../lib/assert"
load "../lib/git"
load "../lib/setup"
load "../lib/wait"
load "../lib/resource"
load "../lib/nomos"

FILE_NAME="$(basename "${BATS_TEST_FILENAME}" '.bats')"

setup() {
  setup::common
}

teardown() {
  setup::common_teardown
}

@test "${FILE_NAME}: Syncs correctly with explicit namespace declarations" {
  if "${RAW_NOMOS}"; then
    kubectl apply -f "${MANIFEST_DIR}/source-format_unstructured.yaml"
    kubectl apply -f "${MANIFEST_DIR}/importer_repoless.yaml"
    nomos::restart_pods
  else
    kubectl apply -f "${MANIFEST_DIR}/operator-config-git-repoless-repo.yaml"
  fi

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

  repair_state
}

@test "${FILE_NAME}: Syncs correctly with only roles" {
  if "${RAW_NOMOS}"; then
    kubectl apply -f "${MANIFEST_DIR}/source-format_unstructured.yaml"
    kubectl apply -f "${MANIFEST_DIR}/importer_repoless-no-ns.yaml"
    nomos::restart_pods
  else
    kubectl apply -f "${MANIFEST_DIR}/operator-config-git-repoless-no-ns.yaml"
  fi

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

  repair_state
}

@test "${FILE_NAME}: Syncs correctly with no policy dir set" {
  if "${RAW_NOMOS}"; then
    kubectl apply -f "${MANIFEST_DIR}/source-format_unstructured.yaml"
    kubectl apply -f "${MANIFEST_DIR}/importer_no-policy-dir.yaml"
    nomos::restart_pods
  else
    kubectl apply -f "${MANIFEST_DIR}/operator-config-git-repoless-no-policy-dir.yaml"
  fi

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

  repair_state
}

function repair_state() {
  if "${RAW_NOMOS}"; then
    kubectl apply -f "${MANIFEST_DIR}/default-configmaps.yaml"
    nomos::restart_pods
  else
    kubectl apply -f "${MANIFEST_DIR}/operator-config-git.yaml"
  fi
}
