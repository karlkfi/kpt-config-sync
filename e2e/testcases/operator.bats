#!/bin/bash

set -euo pipefail

YAML_DIR=${BATS_TEST_DIRNAME}/../testdata
TEST_REPO=${BATS_TMPDIR}/repo

load "../lib/git"
load "../lib/configmap"
load "../lib/setup"
load "../lib/wait"

setup() {
  setup::common
  setup::git::initialize
  setup::git::init_acme
}

teardown() {
  setup::common_teardown
}

@test "ResourceQuota live uninstall/reinstall" {
  # Verify the resourcequota admission controller is currently installed
  wait::for kubectl get deployment -n config-management-system resourcequota-admission-controller
  wait::for kubectl get validatingwebhookconfigurations resource-quota.configmanagement.gke.io

  # Verify that disabling the resource quota admission controller causes it to be uninstalled
  kubectl apply -f "${BATS_TEST_DIRNAME}/../operator-config-git-no-rq.yaml"
  wait::for -f -- kubectl get deployment -n config-management-system resourcequota-admission-controller
  wait::for -f -- kubectl get validatingwebhookconfigurations resource-quota.configmanagement.gke.io

  # Verify that re-enabling the resource quota admission controller causes it to be reinstalled
  kubectl apply -f "${BATS_TEST_DIRNAME}/../operator-config-git.yaml"
  wait::for kubectl get deployment -n config-management-system resourcequota-admission-controller
  wait::for kubectl get validatingwebhookconfigurations resource-quota.configmanagement.gke.io
}

@test "Changing TheNomos target branch takes effect" {
  local configMapPrefix="git-importer"
  # Create a new git branch and make a change there
  cd "${TEST_REPO}"
  echo "git checkout branch"
  git checkout -b branch
  git::update "${YAML_DIR}/branch-rolebinding.yaml" acme/namespaces/eng/backend/bob-rolebinding.yaml
  git commit -m "branch test"
  echo "git push branch"
  git push origin branch -f

  # Apply a Nomos that points to branch "branch"
  kubectl apply -f "${BATS_TEST_DIRNAME}/../operator-config-git-branch.yaml"
  wait::for -t 10 -- configmaps::check_one_exists ${configMapPrefix} config-management-system
  wait::for -t 30 -- kubectl get rolebindings -n backend branch-rolebinding

  # Verify that switching the branch back reverts the branch-rolebinding
  kubectl apply -f "${BATS_TEST_DIRNAME}/../operator-config-git.yaml"
  wait::for -t 10 -- configmaps::check_one_exists ${configMapPrefix} config-management-system
  wait::for -t 30 -f -- kubectl get rolebindings -n backend branch-rolebinding
}
