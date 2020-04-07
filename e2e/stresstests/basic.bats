#!/bin/bash

set -euo pipefail

load "../lib/stress"
load "../lib/setup"

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
  setup::common_teardown
}

@test "${FILE_NAME}: Create 10 namespaces" {
  stress::create_many_namespaces "10"
}

@test "${FILE_NAME}: Create 100 namespaces" {
  stress::create_many_namespaces "100"
}

@test "${FILE_NAME}: Create 1000 namespaces" {
  stress::create_many_namespaces "1000"
}

@test "${FILE_NAME}: Create 10 resources in a single namespace" {
  stress::create_many_resources "10"
}

@test "${FILE_NAME}: Create 100 resources in a single namespace" {
  stress::create_many_resources "100"
}

@test "${FILE_NAME}: Create 500 resources in a single namespace" {
  stress::create_many_resources "500"
}

@test "${FILE_NAME}: Create 100 resources, slow speed" {
  stress::create_many_commits "100" "10"
}

@test "${FILE_NAME}: Create 100 resources, medium speed" {
  stress::create_many_commits "100" "3"
}

@test "${FILE_NAME}: Create 100 resources, fast" {
  stress::create_many_commits "100" "1"
}

@test "${FILE_NAME}: Create 100 resources, RLY fast" {
  stress::create_many_commits "100" "0"
}
