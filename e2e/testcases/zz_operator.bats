#!/bin/bash
# Tests that exercise the per-cluster-addressing functionality.

set -euo pipefail

load "../lib/debug"
load "../lib/git"
load "../lib/setup"
load "../lib/wait"

# Echoes to stdout the current cluster name, based on the content of the configmap
# from the git-policy-importer deployment.
function get_cluster_name() {
  local cluster_name_config_map_name
  cluster_name_config_map_name=$(kubectl get deployments \
    -n=nomos-system git-policy-importer -o yaml \
    --output='jsonpath={.spec.template.spec.containers[0].envFrom[1].configMapRef.name}')
  local cluster_name
  cluster_name=$(kubectl get configmaps \
    -n=nomos-system "${cluster_name_config_map_name}" \
    --output='jsonpath={.data.CLUSTER_NAME}')
  echo "${cluster_name}"
}

# Renames the cluster under test by patching the Nomos resource with the new
# cluster name.  Will fail if the cluster already has that name.
#
# Args:
#   $1: new cluster name (string)
function rename_cluster() {
  local new_name="${1}"
  kubectl patch nomos -n=nomos-system nomos --type=merge \
    -p "{\"spec\":{\"clusterName\": \"${new_name}\"}}"
}

# Renames a cluster under test, and waits until the rename has taken effect.
function expect_rename_to() {
  local expected_cluster_name
  expected_cluster_name="${1}"
  debug::log "Rename cluster to ${expected_cluster_name}"
  wait::for -t 20 -- rename_cluster "${expected_cluster_name}"
  debug::log "Expect cluster name to be ${expected_cluster_name}"
  wait::for -t 40 -o "${expected_cluster_name}" -- get_cluster_name

}

@test "Operator: basic successive cluster renames in Nomos resource" {
  debug::log "Check that the initial cluster state has the correct name"
  wait::for -o "e2e-test-cluster" -- get_cluster_name

  expect_rename_to "eenie"
  expect_rename_to "meenie"
  expect_rename_to "minie"
  expect_rename_to "moe"
}


@test "Operator: Cluster rename load test" {
  skip "Enable this test only if you are debugging"
  debug::log "Check that the initial cluster state has the correct name"
  wait::for -t 40 -o "e2e-test-cluster" -- get_cluster_name

  for count in {0..30}; do
    expect_rename_to "eenie_${count}"
    expect_rename_to "meenie_${count}"
    expect_rename_to "minie_${count}"
    expect_rename_to "moe_${count}"
  done
}

