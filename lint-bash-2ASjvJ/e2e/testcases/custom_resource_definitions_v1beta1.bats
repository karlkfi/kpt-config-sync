#!/bin/bash

set -euo pipefail

load "../lib/debug"
load "../lib/git"
load "../lib/namespaceconfig"
load "../lib/resource"
load "../lib/setup"
load "../lib/wait"

YAML_DIR=${BATS_TEST_DIRNAME}/../testdata

FILE_NAME="$(basename "${BATS_TEST_FILENAME}" '.bats')"

test_setup() {
  setup::git::initialize
  setup::git::commit_minimal_repo_contents

  mkdir acme/cluster
  mkdir -p acme/namespaces/prod

  # Make sure tests start with a clean slate of no CRDs on the cluster.
  kubectl delete crd anvils.acme.com --ignore-not-found
  kubectl delete crd clusteranvils.acme.com --ignore-not-found
}

test_teardown() {
  setup::git::remove_all acme
}

test_-24-7bFILE-5fNAME-7d-3a_CRD_added-2c_CR_added-2c_CRD_removed-2c_and_then_CR_removed_from_repo() { bats_test_begin "${FILE_NAME}: CRD added, CR added, CRD removed, and then CR removed from repo" 32; 
  debug::log "Adding CRD"
  git::add "${YAML_DIR}/customresources/v1beta1_crds/anvil-crd.yaml" acme/cluster/anvil-crd.yaml
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

test_-24-7bFILE-5fNAME-7d-3a_namespace-2dscoped_CRD_and_CR_added-2c_CRD_updated-2c_and_both_removed_from_repo() { bats_test_begin "${FILE_NAME}: namespace-scoped CRD and CR added, CRD updated, and both removed from repo" 63; 
  debug::log "Adding CRD and Custom Resource"
  git::add "${YAML_DIR}/customresources/v1beta1_crds/anvil-crd.yaml" acme/cluster/anvil-crd.yaml
  git::add "${YAML_DIR}/customresources/anvil.yaml" acme/namespaces/prod/anvil.yaml
  namespace::declare prod
  git::commit

  debug::log "Custom Resource exists on cluster"
  # we need to specify the version of anvil in kubectl. Otherwise, it may try to get a version that doesn't exist.
  wait::for -t 30 -- kubectl get anvil.v1.acme.com e2e-test-anvil -n prod

  debug::log "CRD exists on cluster"
  wait::for -t 30 -- kubectl get crd anvils.acme.com

  debug::log "Updating CRD version"
  git::update "${YAML_DIR}/customresources/v1beta1_crds/anvil-crd-v2.yaml" acme/cluster/anvil-crd.yaml
  git::commit

  debug::log "CRD on cluster has v2 as stored version"
  wait::for -t 30 -- nomos::repo_synced

  svname=$(kubectl get crd anvils.acme.com --output='jsonpath={.spec.versions[1].name}')
  if [[ ${svname} != "v2" ]]; then
    debug::error "Want versions[1].name = v2 but got $svname"
  fi
  svserved=$(kubectl get crd anvils.acme.com --output='jsonpath={.spec.versions[1].served}')
  if [[ ${svserved} != "true" ]]; then
    debug::error "Want versions[1].served = true but got $svserved"
  fi
  svstorage=$(kubectl get crd anvils.acme.com --output='jsonpath={.spec.versions[1].storage}')
  if [[ ${svstorage} != "true" ]]; then
    debug::error "Want versions[1].storage = true but got $svstorage"
  fi

  debug::log "Updating CRD to only support version v2 and update CR to v2"
  git::update "${YAML_DIR}/customresources/v1beta1_crds/anvil-crd-only-v2.yaml" acme/cluster/anvil-crd.yaml
  git::update "${YAML_DIR}/customresources/anvil-heavier-v2.yaml" acme/namespaces/prod/anvil.yaml
  git::commit

  debug::log "CR is updated"
  # we need to specify the version of anvil in kubectl. Otherwise, it may try to get a version that doesn't exist.
  wait::for -t 30 -o "100" -- \
    kubectl get anvils.v2.acme.com -n prod e2e-test-anvil \
      --output='jsonpath={.spec.lbs}'

  debug::log "CRD on cluster doesn't serve or store v1 Anvils"
  wait::for -t 30 -- nomos::repo_synced

  svname=$(kubectl get crd anvils.acme.com --output='jsonpath={.spec.versions[0].name}')
  if [[ ${svname} != "v1" ]]; then
    debug::error "Want versions[0].name = v1 but got $svname"
  fi
  svserved=$(kubectl get crd anvils.acme.com --output='jsonpath={.spec.versions[0].served}')
  if [[ ${svserved} != "false" ]]; then
    debug::error "Want versions[0].served = false but got $svserved"
  fi
  svstorage=$(kubectl get crd anvils.acme.com --output='jsonpath={.spec.versions[0].storage}')
  if [[ ${svstorage} != "false" ]]; then
    debug::error "Want versions[0].storage = false but got $svstorage"
  fi

  debug::log "Removing CRD and Custom Resource"
  git::rm acme/cluster/anvil-crd.yaml
  git::rm acme/namespaces/prod/anvil.yaml
  git::commit

  debug::log "Custom Resource doesn't exist on cluster"
  wait::for -f -t 30 -- kubectl get anvil e2e-test-anvil -n prod

  debug::log "CRD doesn't exist on cluster"
  wait::for -f -t 30 -- kubectl get crd anvils.acme.com
}

test_-24-7bFILE-5fNAME-7d-3a_cluster-2dscoped_CRD_and_CR_added-2c_CRD_updated-2c_and_both_removed_from_repo() { bats_test_begin "${FILE_NAME}: cluster-scoped CRD and CR added, CRD updated, and both removed from repo" 136; 
  debug::log "Adding clusterrole to avoid importer safety check"
  git::add "${YAML_DIR}/clusterrole-create.yaml" acme/cluster/e2e-test-clusterrole.yaml
  git::commit

  debug::log "Adding CRD and Custom Resource"
  git::add "${YAML_DIR}/customresources/v1beta1_crds/clusteranvil-crd.yaml" acme/cluster/clusteranvil-crd.yaml
  git::add "${YAML_DIR}/customresources/clusteranvil.yaml" acme/cluster/clusteranvil.yaml
  git::commit

  debug::log "Custom Resource exists on cluster"
  wait::for -t 30 -- kubectl get clusteranvil e2e-test-clusteranvil

  debug::log "CRD exists on cluster"
  wait::for -t 30 -- kubectl get crd clusteranvils.acme.com

  debug::log "Updating CRD version"
  git::update "${YAML_DIR}/customresources/v1beta1_crds/clusteranvil-crd-v2.yaml" acme/cluster/clusteranvil-crd.yaml
  git::commit

  debug::log "CRD on cluster has v2 as stored version"
  wait::for -t 30 -- nomos::repo_synced

  svname=$(kubectl get crd clusteranvils.acme.com --output='jsonpath={.spec.versions[1].name}')
  if [[ ${svname} != "v2" ]]; then
    debug::error "Want versions[1].name = v2 but got $svname"
  fi
  svserved=$(kubectl get crd clusteranvils.acme.com --output='jsonpath={.spec.versions[1].served}')
  if [[ ${svserved} != "true" ]]; then
    debug::error "Want versions[1].served = true but got $svserved"
  fi
  svstorage=$(kubectl get crd clusteranvils.acme.com --output='jsonpath={.spec.versions[1].storage}')
  if [[ ${svstorage} != "true" ]]; then
    debug::error "Want versions[1].storage = true but got $svstorage"
  fi

  debug::log "Removing CRD and Custom Resource"
  git::rm acme/cluster/clusteranvil-crd.yaml
  git::rm acme/cluster/clusteranvil.yaml
  git::commit

  debug::log "Custom Resource doesn't exist on cluster"
  wait::for -f -t 30 -- kubectl get clusteranvil e2e-test-clusteranvil

  debug::log "CRD doesn't exist on cluster"
  wait::for -f -t 30 -- kubectl get crd clusteranvils.acme.com
}

test_-24-7bFILE-5fNAME-7d-3a_CRD_added-2c_unmanaged_CR_added-2c_and_CRD_removed_which_removes_CR_from_cluster() { bats_test_begin "${FILE_NAME}: CRD added, unmanaged CR added, and CRD removed which removes CR from cluster" 184; 
  debug::log "Adding CRD"
  git::add "${YAML_DIR}/customresources/v1beta1_crds/anvil-crd.yaml" acme/cluster/anvil-crd.yaml
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

test_-24-7bFILE-5fNAME-7d-3a_CRD_added-2c_CR_added-2c_CRD_removed-2c_which_causes_vet_failure() { bats_test_begin "${FILE_NAME}: CRD added, CR added, CRD removed, which causes vet failure" 209; 
  debug::log "Adding CRD"
  git::add "${YAML_DIR}/customresources/v1beta1_crds/anvil-crd.yaml" acme/cluster/anvil-crd.yaml
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
  wait::for -f -t 5 -- nomos vet --path="${TEST_REPO}"
}

bats_test_function test_-24-7bFILE-5fNAME-7d-3a_CRD_added-2c_CR_added-2c_CRD_removed-2c_and_then_CR_removed_from_repo
bats_test_function test_-24-7bFILE-5fNAME-7d-3a_namespace-2dscoped_CRD_and_CR_added-2c_CRD_updated-2c_and_both_removed_from_repo
bats_test_function test_-24-7bFILE-5fNAME-7d-3a_cluster-2dscoped_CRD_and_CR_added-2c_CRD_updated-2c_and_both_removed_from_repo
bats_test_function test_-24-7bFILE-5fNAME-7d-3a_CRD_added-2c_unmanaged_CR_added-2c_and_CRD_removed_which_removes_CR_from_cluster
bats_test_function test_-24-7bFILE-5fNAME-7d-3a_CRD_added-2c_CR_added-2c_CRD_removed-2c_which_causes_vet_failure
