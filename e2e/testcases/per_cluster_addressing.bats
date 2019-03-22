#!/bin/bash
# Tests that exercise the per-cluster-addressing functionality.

set -euo pipefail

load "../lib/debug"
load "../lib/configmap"
load "../lib/git"
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

YAML_DIR=${BATS_TEST_DIRNAME}/../testdata

function add_clusterregistry_data() {
  debug::log "Add cluster, and cluster registry data"
  git::add \
    "${YAML_DIR}/clusterregistry-cluster-env-prod.yaml" \
    acme/clusterregistry/cluster-1.yaml
  git::add \
    "${YAML_DIR}/clusterselector-env-prod.yaml" \
    acme/clusterregistry/clusterselector-1.yaml
  git::add \
    "${YAML_DIR}/clusterselector-env-other.yaml" \
    acme/clusterregistry/clusterselector-2.yaml
  git::commit -m "Add cluster and cluster registry data"
}

# Echoes to stdout the current cluster name, based on the content of the configmap
# from the git-importer deployment.
function get_cluster_name() {
  local cluster_name_config_map_name
  cluster_name_config_map_name=$(kubectl get deployments \
    -n=config-management-system git-importer -o yaml \
    --output='jsonpath={.spec.template.spec.containers[0].envFrom[1].configMapRef.name}')
  local cluster_name
  cluster_name=$(kubectl get configmaps \
    -n=config-management-system "${cluster_name_config_map_name}" \
    --output='jsonpath={.data.CLUSTER_NAME}')
  echo "${cluster_name}"
}

@test "ClusterSelector: Target different ResourceQuotas to different clusters" {
  add_clusterregistry_data

  debug::log "Ensure that the cluster has expected name at the beginning"
  local expected_cluster_name="e2e-test-cluster"
  wait::for -o "${expected_cluster_name}" -- get_cluster_name

  debug::log "Add a valid cluster selector annotation to a role binding"
  git::update \
    "${YAML_DIR}/addressed-quota.yaml" \
    acme/namespaces/eng/quota.yaml
  git::commit -m "Adding quotas targeted to different clusters"

  debug::log "Wait for resource quota to appear in the namespace frontend"
  local magic_pod_count="133"  # Hard-coded in addressed-quota.yaml
  wait::for -o "${magic_pod_count}" -- \
    kubectl get resourcequotas --namespace=frontend \
    config-management-resource-quota --output='jsonpath={.spec.hard.pods}'
}

@test "ClusterSelector: Object reacts to per-cluster annotations" {
  debug::log "Require that the cluster name exists on the cluster"
  wait::for -t 10 -- kubectl get configmaps -n config-management-system cluster-name

  add_clusterregistry_data

  debug::log "Adding a valid cluster selector annotation to a role binding"
  git::update \
    "${YAML_DIR}/backend/bob-rolebinding-env-prod.yaml" \
    acme/namespaces/eng/backend/bob-rolebinding.yaml
  git::commit -m "Add a valid cluster selector annotation to a role binding"

  debug::log "Wait for bob-rolebinding to appear in the namespace backend"
  wait::for -- kubectl get rolebindings -n backend bob-rolebinding

  debug::log "Change cluster selector to selector-env-test"
  git::update \
    "${YAML_DIR}/backend/bob-rolebinding-env-test.yaml" \
    acme/namespaces/eng/backend/bob-rolebinding.yaml
  git::commit -m "Change cluster selector to selector-env-test"

  debug::log "Wait for bob-rolebinding to disappear in namespace backend"
  wait::for -f -- kubectl get rolebindings -n backend bob-rolebinding

  debug::log "Revert to selector-env-prod"
  git::update \
    "${YAML_DIR}/backend/bob-rolebinding-env-prod.yaml" \
    acme/namespaces/eng/backend/bob-rolebinding.yaml
  git::commit -m "Revert to selector-env-prod"

  debug::log "Wait for bob-rolebinding to reappear in the backend namespace"
  wait::for -- kubectl get rolebindings -n backend bob-rolebinding
}

@test "ClusterSelector: Namespace reacts to per-cluster annotations" {
  add_clusterregistry_data

  debug::log "Adding a valid cluster selector annotation to a namespace"
  git::update \
    "${YAML_DIR}/backend/namespace-env-prod.yaml" \
    acme/namespaces/eng/backend/namespace.yaml
  git::commit -m "Add a valid cluster selector annotation to a namespace"

  debug::log "Wait for namespace backend to be there"
  wait::for -- kubectl get ns backend
  debug::log "Wait for bob-rolebinding to appear in the namespace backend"
  wait::for -- kubectl get rolebindings -n backend bob-rolebinding

  debug::log "Change cluster selector to selector-env-test"
  git::update \
    "${YAML_DIR}/backend/namespace-env-test.yaml" \
    acme/namespaces/eng/backend/namespace.yaml
  git::commit -m "Change cluster selector to selector-env-test"

  debug::log "Wait for bob-rolebinding to disappear in namespace backend"
  wait::for -f -- kubectl get rolebindings -n backend bob-rolebinding
  debug::log "Wait for namespace backend to disappear"
  wait::for -f -- kubectl get ns backend

  debug::log "Revert to selector-env-prod"
  git::update \
    "${YAML_DIR}/backend/namespace-env-prod.yaml" \
    acme/namespaces/eng/backend/namespace.yaml
  git::commit -m "Revert to selector-env-prod"

  debug::log "Wait for namespace backend to reappear"
  wait::for -- kubectl get ns backend
  debug::log "Wait for bob-rolebinding to reapear in the backend namespace"
  wait::for -- kubectl get rolebindings -n backend bob-rolebinding
}

@test "ClusterSelector: Object reacts to changing the selector" {
  add_clusterregistry_data

  debug::log "Adding a valid cluster selector annotation to a role binding"
  git::update \
    "${YAML_DIR}/backend/bob-rolebinding-env-prod.yaml" \
    acme/namespaces/eng/backend/bob-rolebinding.yaml
  git::commit -m "Add a valid cluster selector annotation to a role binding"

  debug::log "Wait for bob-rolebinding to appear in the namespace backend"
  wait::for -- kubectl get rolebindings -n backend bob-rolebinding

  debug::log "Modify the cluster selector to select a different environment"
  git::update \
    "${YAML_DIR}/clusterselector-env-test.yaml" \
    acme/clusterregistry/clusterselector-1.yaml
  git::commit -m "Modify the cluster selector to select a different environment"

  debug::log "Wait for bob-rolebinding to disappear from the namespace backend"
  wait::for -f -- kubectl get rolebindings -n backend bob-rolebinding
}

@test "ClusterSelector: Cluster reacts to CLUSTER_NAME change" {
  debug::log "Require that the cluster name exists on the cluster"
  wait::for -t 10 -- kubectl get configmaps -n config-management-system cluster-name

  add_clusterregistry_data

  debug::log "Adding a valid cluster selector annotation to a role binding"
  git::update \
    "${YAML_DIR}/backend/bob-rolebinding-env-prod.yaml" \
    acme/namespaces/eng/backend/bob-rolebinding.yaml
  git::commit -m "Add a valid cluster selector annotation to a role binding"

  debug::log "Wait for bob-rolebinding to appear in the namespace backend"
  wait::for -- kubectl get rolebindings -n backend bob-rolebinding

  debug::log "Change cluster selector to selector-env-test"
  git::update \
    "${YAML_DIR}/backend/bob-rolebinding-env-test.yaml" \
    acme/namespaces/eng/backend/bob-rolebinding.yaml
  git::commit -m "Change cluster selector to selector-env-test"

  debug::log "Wait for bob-rolebinding to disappear in namespace backend"
  wait::for -f -- kubectl get rolebindings -n backend bob-rolebinding

  debug::log "Change the cluster name"
  wait::for -- kubectl patch configmanagement -n=config-management-system config-management --type=merge \
    -p '{"spec":{"clusterName": "test-cluster-env-test"}}'

  debug::log "Change the cluster and selector to point to test"
  git::update \
    "${YAML_DIR}/clusterregistry-cluster-env-test.yaml" \
    acme/clusterregistry/cluster-1.yaml
  git::update \
    "${YAML_DIR}/clusterselector-env-test.yaml" \
    acme/clusterregistry/clusterselector-1.yaml
  git::commit -m "Change cluster selector to selector-env-test"

  debug::log "Wait for bob-rolebinding to reappear in the backend namespace"
  wait::for -- kubectl get rolebindings -n backend bob-rolebinding
}

# Renames the cluster under test by patching the Nomos resource with the new
# cluster name.  Will fail if the cluster already has that name.
#
# Args:
#   $1: new cluster name (string)
function rename_cluster() {
  local new_name="${1}"
  kubectl patch configmanagement -n=config-management-system config-management --type=merge \
    -p "{\"spec\":{\"clusterName\": \"${new_name}\"}}"
}

# Renames a cluster under test, and waits until the rename has taken effect.
function expect_rename_to() {
  local configMapPrefix="cluster-name"
  local expected_cluster_name
  expected_cluster_name="${1}"
  debug::log "Rename cluster to ${expected_cluster_name}"
  wait::for -t 20 -- rename_cluster "${expected_cluster_name}"
  debug::log "Expect cluster name to be ${expected_cluster_name}"
  wait::for -t 40 -o "${expected_cluster_name}" -- get_cluster_name
  wait::for -t 10 -- configmaps::check_one_exists ${configMapPrefix} config-management-system
}

@test "Operator: Cluster rename load test" {
  for count in {0..3}; do
    expect_rename_to "eenie_${count}"
    expect_rename_to "meenie_${count}"
    expect_rename_to "minie_${count}"
    expect_rename_to "moe_${count}"
  done
}
