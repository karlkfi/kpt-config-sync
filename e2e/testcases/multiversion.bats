#!/bin/bash

set -euo pipefail

load "../lib/git"
load "../lib/setup"
load "../lib/wait"

FILE_NAME="$(basename "${BATS_TEST_FILENAME}" '.bats')"

setup() {
  setup::common
  setup::git::initialize
  setup::git::init acme
}

teardown() {
  setup::git::remove_all acme
  setup::common_teardown
}

YAML_DIR=${BATS_TEST_DIRNAME}/../testdata/multiversion

@test "${FILE_NAME}: Multiple versions of RoleBindings" {
  git::add "${YAML_DIR}/v1beta1.yaml" acme/namespaces/eng/backend/v1beta1.yaml
  git::commit

  wait::for -t 30 -- kubectl get rolebinding -n backend v1beta1user

  git::add "${YAML_DIR}/v1.yaml" acme/namespaces/eng/backend/v1.yaml
  git::commit

  wait::for -t 30 -- kubectl get rolebinding -n backend v1user
  kubectl get rolebinding -n backend v1beta1user

  git::rm acme/namespaces/eng/backend/v1beta1.yaml
  git::commit

  wait::for -f -t 30 -- kubectl get rolebinding -n backend v1beta1user
  kubectl get rolebinding -n backend v1user

  git::rm acme/namespaces/eng/backend/v1.yaml
  git::commit

  wait::for -f -t 30 -- kubectl get rolebinding -n backend v1user
}
