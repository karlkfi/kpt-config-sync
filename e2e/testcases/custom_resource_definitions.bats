#!/bin/bash

set -euo pipefail

load "../lib/debug"
load "../lib/git"
load "../lib/namespaceconfig"
load "../lib/resource"
load "../lib/setup"
load "../lib/wait"

YAML_DIR=${BATS_TEST_DIRNAME}/../testdata

setup() {
  setup::common
  setup::git::initialize

  local TEST_REPO_DIR=${BATS_TMPDIR}
  cd "${TEST_REPO_DIR}/repo"

  mkdir -p acme/system
  cp -r /opt/testing/e2e/examples/acme/system acme

  mkdir acme/cluster
  mkdir -p acme/namespaces/prod
  git add -A
  git::commit -a -m "Commit minimal repo contents."

  # Make sure tests start with a clean slate of no CRDs on the cluster.
  wait::for -f -t 30 -- kubectl get crd anvils.acme.com
  wait::for -f -t 30 -- kubectl get crd clusteranvils.acme.com
}

function teardown() {
  setup::common_teardown
}

@test "CRD added, CR added, CRD removed, and then CR removed from repo" {
  debug::log "Adding CRD"
  git::add "${YAML_DIR}/customresources/anvil-crd.yaml" acme/cluster/anvil-crd.yaml
  git::commit

  debug::log "CRD exists on cluster"
  wait::for -t 30 -- kubectl get crd anvils.acme.com

  debug::log "Adding Custom Resource"
  git::add "${YAML_DIR}/customresources/anvil.yaml" acme/namespaces/prod/anvil.yaml
  namespace::declare prod
  git::commit

  debug::log "Custom Resource exists on cluster"
  wait::for -t 30 -- kubectl get anvil e2e-test-anvil -n prod

  debug::log "Removing Custom Resource"
  git::rm acme/namespaces/prod/anvil.yaml
  git::commit

  debug::log "Custom Resource doesn't exists on cluster"
  wait::for -f -t 30 -- kubectl get anvil e2e-test-anvil -n prod

  debug::log "Removing CRD"
  git::rm acme/cluster/anvil-crd.yaml
  git::commit

  debug::log "CRD doesn't exist on cluster"
  wait::for -f -t 30 -- kubectl get crd anvils.acme.com
}

@test "namespace-scoped CRD and CR added, CRD updated, and both removed from repo" {
  debug::log "Adding CRD and Custom Resource"
  git::add "${YAML_DIR}/customresources/anvil-crd.yaml" acme/cluster/anvil-crd.yaml
  git::add "${YAML_DIR}/customresources/anvil.yaml" acme/namespaces/prod/anvil.yaml
  namespace::declare prod
  git::commit

  debug::log "Custom Resource exists on cluster"
  # we need to specify the version of anvil in kubectl. Otherwise, it may try to get a version that doesn't exist.
  wait::for -t 30 -- kubectl get anvil.v1.acme.com e2e-test-anvil -n prod

  debug::log "CRD exists on cluster"
  wait::for -t 30 -- kubectl get crd anvils.acme.com

  debug::log "Updating CRD version"
  git::update "${YAML_DIR}/customresources/anvil-crd-v2.yaml" acme/cluster/anvil-crd.yaml
  git::commit

  debug::log "CRD on cluster has v2 as stored version"
  wait::for -t 30 -o "map[name:v2 served:true storage:true]" -- \
    kubectl get crd anvils.acme.com \
      --output='jsonpath={.spec.versions[1]}' # v2 is the second element in versions

  debug::log "Updating CRD to only support version v2 and update CR to v2"
  git::update "${YAML_DIR}/customresources/anvil-crd-only-v2.yaml" acme/cluster/anvil-crd.yaml
  git::update "${YAML_DIR}/customresources/anvil-heavier-v2.yaml" acme/namespaces/prod/anvil.yaml
  git::commit

  debug::log "CR is updated"
  # we need to specify the version of anvil in kubectl. Otherwise, it may try to get a version that doesn't exist.
  wait::for -t 30 -o "100" -- \
    kubectl get anvils.v2.acme.com -n prod e2e-test-anvil \
      --output='jsonpath={.spec.lbs}'

  debug::log "CRD on cluster doesn't serve or store v1 Anvils"
  wait::for -t 30 -o "map[name:v1 served:false storage:false]" -- \
    kubectl get crd anvils.acme.com \
      --output='jsonpath={.spec.versions[0]}' # v1 is the first element in versions

  debug::log "Removing CRD and Custom Resource"
  git::rm acme/cluster/anvil-crd.yaml
  git::rm acme/namespaces/prod/anvil.yaml
  git::commit

  debug::log "Custom Resource doesn't exist on cluster"
  wait::for -f -t 30 -- kubectl get anvil e2e-test-anvil -n prod

  debug::log "CRD doesn't exist on cluster"
  wait::for -f -t 30 -- kubectl get crd anvils.acme.com
}

@test "cluster-scoped CRD and CR added, CRD updated, and both removed from repo" {
  debug::log "Adding CRD and Custom Resource"
  git::add "${YAML_DIR}/customresources/clusteranvil-crd.yaml" acme/cluster/clusteranvil-crd.yaml
  git::add "${YAML_DIR}/customresources/clusteranvil.yaml" acme/cluster/clusteranvil.yaml
  git::commit

  debug::log "Custom Resource exists on cluster"
  wait::for -t 30 -- kubectl get clusteranvil e2e-test-clusteranvil

  debug::log "CRD exists on cluster"
  wait::for -t 30 -- kubectl get crd clusteranvils.acme.com

  debug::log "Updating CRD version"
  git::update "${YAML_DIR}/customresources/clusteranvil-crd-v2.yaml" acme/cluster/clusteranvil-crd.yaml
  git::commit

  debug::log "CRD on cluster has v2 as stored version"
  wait::for -t 30 -o "map[name:v2 served:true storage:true]" -- \
    kubectl get crd clusteranvils.acme.com \
      --output='jsonpath={.spec.versions[1]}' # v2 is the second element in versions

  debug::log "Removing CRD and Custom Resource"
  git::rm acme/cluster/clusteranvil-crd.yaml
  git::rm acme/cluster/clusteranvil.yaml
  git::commit

  debug::log "Custom Resource doesn't exist on cluster"
  wait::for -f -t 30 -- kubectl get clusteranvil e2e-test-clusteranvil

  debug::log "CRD doesn't exist on cluster"
  wait::for -f -t 30 -- kubectl get crd clusteranvils.acme.com
}

@test "CRD added, unmanaged CR added, and CRD removed which removes CR from cluster" {
  debug::log "Adding CRD"
  git::add "${YAML_DIR}/customresources/anvil-crd.yaml" acme/cluster/anvil-crd.yaml
  git::commit

  debug::log "CRD exists on cluster"
  wait::for -t 30 -- kubectl get crd anvils.acme.com

  debug::log "Adding unmanaged CR"
  kubectl apply -f  "${YAML_DIR}/customresources/anvil-heavier.yaml"

  debug::log "unmanaged CR exists on cluster"
  wait::for -t 30 -- kubectl get anvils e2e-test-anvil -n default

  debug::log "Removing CRD"
  git::rm acme/cluster/anvil-crd.yaml
  git::commit

  debug::log "CRD doesn't exist on cluster"
  wait::for -f -t 30 -- kubectl get crd anvils.acme.com

  debug::log "unmanaged CRD doesn't exist on cluster"
  wait::for -f -t 30 -- kubectl get anvils e2e-test-anvil -n default
}

@test "CRD added, CR added, CR removed, which causes vet failure" {
  debug::log "Adding CRD"
  git::add "${YAML_DIR}/customresources/anvil-crd.yaml" acme/cluster/anvil-crd.yaml
  git::commit

  debug::log "CRD exists on cluster"
  wait::for -t 30 -- kubectl get crd anvils.acme.com

  debug::log "Adding Custom Resource"
  git::add "${YAML_DIR}/customresources/anvil.yaml" acme/namespaces/prod/anvil.yaml
  namespace::declare prod
  git::commit

  debug::log "Custom Resource exists on cluster"
  wait::for -t 30 -- kubectl get anvil e2e-test-anvil -n prod

  debug::log "Removing CRD"
  git::rm acme/cluster/anvil-crd.yaml
  git::commit

  debug::log "Check that nomos vet fails"
  wait::for -f -t 5 -- /opt/testing/go/bin/linux_amd64/nomos vet --path="${BATS_TMPDIR}/repo"
}
