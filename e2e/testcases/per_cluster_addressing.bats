#!/bin/bash
# Tests that exercise the per-cluster-addressing functionality.

set -euo pipefail

load "../lib/debug"
load "../lib/git"
load "../lib/setup"
load "../lib/wait"

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

@test "ClusterSelector: Object reacts to per-cluster annotations" {
  debug::log "Require that the cluster name exists on the cluster"
  wait::for -t 10 -- kubectl get configmaps -n nomos-system cluster-name

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
  wait::for -t 10 -- kubectl get configmaps -n nomos-system cluster-name

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
  wait::for -- kubectl patch nomos -n=nomos-system nomos --type=merge \
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

  debug::log "Change the cluster name back to what it used to be"
  wait::for -- kubectl patch nomos -n=nomos-system nomos --type=merge \
    -p '{"spec":{"clusterName": "e2e-test-cluster"}}'
}
