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
  kubectl apply -f "${BATS_TEST_DIRNAME}/../operator-config-git-repoless-repo.yaml"
  setup::git::initialize
  setup::git::init_acme repoless

  wait::for -t 60 -- nomos::repo_synced

  resource::check clusterrole repoless-admin
  resource::check namespace backend
  resource::check role pod-reader-backend -n backend
  resource::check namespace default
  resource::check role pod-reader-default -n default

  # Repair the state
  kubectl apply -f "${BATS_TEST_DIRNAME}/../operator-config-git.yaml"
}

@test "${FILE_NAME}: Syncs correctly with only roles" {
  kubectl apply -f "${BATS_TEST_DIRNAME}/../operator-config-git-repoless-no-ns.yaml"
  setup::git::initialize
  setup::git::init_acme repoless-no-ns

  wait::for -t 60 -- nomos::repo_synced

  resource::check clusterrole repoless-admin
  resource::check namespace backend
  resource::check role pod-reader-backend -n backend
  resource::check namespace default
  resource::check role pod-reader-default -n default

  # Repair the state
  kubectl apply -f "${BATS_TEST_DIRNAME}/../operator-config-git.yaml"
}
