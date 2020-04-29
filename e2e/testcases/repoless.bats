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
    kubectl apply -f "${MANIFEST_DIR}/repoless-repo.yaml"
  else
    kubectl apply -f "${MANIFEST_DIR}/operator-config-git-repoless-repo.yaml"
  fi

  setup::git::initialize
  setup::git::init repoless

  wait::for -t 60 -- nomos::repo_synced

  resource::check clusterrole repoless-admin
  resource::check namespace backend
  resource::check role pod-reader-backend -n backend
  resource::check namespace default
  resource::check role pod-reader-default -n default

  repair_state
}

@test "${FILE_NAME}: Syncs correctly with only roles" {
  if "${RAW_NOMOS}"; then
    kubectl apply -f "${MANIFEST_DIR}/repoless-no-ns.yaml"
  else
    kubectl apply -f "${MANIFEST_DIR}/operator-config-git-repoless-no-ns.yaml"
  fi

  setup::git::initialize
  setup::git::init repoless-no-ns

  wait::for -t 60 -- nomos::repo_synced

  resource::check clusterrole repoless-admin
  resource::check namespace backend
  resource::check role pod-reader-backend -n backend
  resource::check namespace default
  resource::check role pod-reader-default -n default

  repair_state
}

@test "${FILE_NAME}: Syncs correctly with no policy dir set" {
  if "${RAW_NOMOS}"; then
    kubectl apply -f "${MANIFEST_DIR}/repoless-no-policy-dir.yaml"
  else
    kubectl apply -f "${MANIFEST_DIR}/operator-config-git-repoless-no-policy-dir.yaml"
  fi

  setup::git::initialize
  setup::git::init repoless

  wait::for -t 60 -- nomos::repo_synced

  resource::check clusterrole repoless-admin
  resource::check namespace backend
  resource::check role pod-reader-backend -n backend
  resource::check namespace default
  resource::check role pod-reader-default -n default

  repair_state
}

function repair_state() {
  if "${RAW_NOMOS}"; then
    kubectl apply -f "${MANIFEST_DIR}/base-configmaps.yaml"
  else
    kubectl apply -f "${MANIFEST_DIR}/operator-config-git.yaml"
  fi
}
