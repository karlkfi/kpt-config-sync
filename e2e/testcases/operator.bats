#!/bin/bash

set -euo pipefail

YAML_DIR=${BATS_TEST_DIRNAME}/../testdata
TEST_REPO=${BATS_TMPDIR}/repo

load "../lib/assert"
load "../lib/cluster"
load "../lib/debug"
load "../lib/git"
load "../lib/ignore"
load "../lib/namespace"
load "../lib/policynode"
load "../lib/resource"
load "../lib/setup"
load "../lib/wait"

@test "ResourceQuota live uninstall/reinstall" {
  # Verify the resourcequota admission controller is currently installed
  wait::for kubectl get deployment -n nomos-system resourcequota-admission-controller
  wait::for kubectl get validatingwebhookconfigurations resource-quota.nomos.dev

  # Verify that disabling the resource quota admission controller causes it to be uninstalled
  kubectl apply -f "${BATS_TEST_DIRNAME}/../operator-config-git-no-rq.yaml"
  wait::for -f -- kubectl get deployment -n nomos-system resourcequota-admission-controller
  wait::for -f -- kubectl get validatingwebhookconfigurations resource-quota.nomos.dev

  # Verify that re-enabling the resource quota admission controller causes it to be reinstalled
  kubectl apply -f "${BATS_TEST_DIRNAME}/../operator-config-git.yaml"
  wait::for kubectl get deployment -n nomos-system resourcequota-admission-controller
  wait::for kubectl get validatingwebhookconfigurations resource-quota.nomos.dev
}

@test "Changing TheNomos target branch takes effect" {
  # Create a new git branch and make a change there
  cd "${TEST_REPO}"
  echo "git checkout branch"
  git checkout -b branch
  git::update "${YAML_DIR}/branch-rolebinding.yaml" acme/namespaces/eng/backend/bob-rolebinding.yaml
  git commit -m "branch test"
  echo "git push branch"
  git push origin branch

  # Apply a Nomos that points to branch "branch"
  kubectl apply -f "${BATS_TEST_DIRNAME}/../operator-config-git-branch.yaml"

  wait::for -t 30 -- kubectl get rolebindings -n backend branch-rolebinding

  # Verify that switching the branch back reverts the branch-rolebinding
  kubectl apply -f "${BATS_TEST_DIRNAME}/../operator-config-git.yaml"
  wait::for -t 30 -f -- kubectl get rolebindings -n backend branch-rolebinding
}
