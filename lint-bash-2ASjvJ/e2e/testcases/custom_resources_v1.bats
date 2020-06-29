#!/bin/bash

set -euo pipefail

load "../lib/debug"
load "../lib/git"
load "../lib/namespaceconfig"
load "../lib/resource"
load "../lib/setup"
load "../lib/wait"

YAML_DIR=${BATS_TEST_DIRNAME}/../testdata
WATCH_PID=""

FILE_NAME="$(basename "${BATS_TEST_FILENAME}" '.bats')"

MINOR_VERSION="$(kubectl version -ojson | jq -r '.serverVersion.minor[0:2]')"

test_setup() {
  setup::git::initialize
  setup::git::init acme
}

# This cleans up any CRDs that were created by a testcase
test_teardown() {
  kubectl delete crd anvils.acme.com --ignore-not-found=true || true
  kubectl delete crd clusteranvils.acme.com --ignore-not-found=true || true
  wait::for -f -t 30 -- kubectl get crd anvils.acme.com
  wait::for -f -t 30 -- kubectl get crd clusteranvils.acme.com
  if [[ "$WATCH_PID" != "" ]]; then
    kill $WATCH_PID || true
    WATCH_PID=""
  fi
  setup::git::remove_all acme
}

test_-24-7bFILE-5fNAME-7d-3a_CRD_deleted_before_repo_update() { bats_test_begin "${FILE_NAME}: CRD deleted before repo update" 37; 
  if (( MINOR_VERSION < 16 )); then
    skip
  fi

  local resname="e2e-test-anvil"
  kubectl apply -f "${YAML_DIR}/customresources/v1_crds/anvil-crd.yaml"
  kubectl apply -f "${YAML_DIR}/customresources/v1_crds/clusteranvil-crd.yaml"
  resource::check crd anvils.acme.com
  resource::check crd clusteranvils.acme.com

  debug::log "Creating custom resource"
  git::add "${YAML_DIR}/customresources/anvil.yaml" acme/namespaces/eng/backend/anvil.yaml
  git::commit

  debug::log "Custom resource exists on cluster"
  wait::for -t 30 -- kubectl get anvil ${resname} -n backend

  debug::log "Removing CRD for custom resource"
  kubectl delete crd anvils.acme.com --ignore-not-found=true || true

  debug::log "Removing anvil from repo"
  git::rm acme/namespaces/eng/backend/anvil.yaml
  git::commit

  debug::log "Waiting for sync removal"
  wait::for -f -t 10 -- kubectl get sync anvils.acme.com
}

test_-24-7bFILE-5fNAME-7d-3a_Sync_and_update_structural_custom_resource() { bats_test_begin "${FILE_NAME}: Sync and update structural custom resource" 66; 
  if (( MINOR_VERSION < 16 )); then
    skip
  fi

  local resname="e2e-test-anvil"

  debug::log "Adding CRD out of band"
  run kubectl apply -f "${YAML_DIR}/customresources/v1_crds/anvil-crd-structural.yaml"

  # shellcheck disable=SC2154
  if [[ "$status" -eq 1 ]]; then
    # TODO(b/149864521): Remove this conditional skip once we no longer support 1.14 or below.
    debug::log "Unable to add structural CRD to cluster. This is expected for k8s < 1.15"
    skip
  fi

  resource::check crd anvils.acme.com

  debug::log "Updating repo with custom resource"
  git::add "${YAML_DIR}/customresources/anvil.yaml" acme/namespaces/eng/backend/anvil.yaml
  git::commit

  debug::log "Checking that custom resource appears on cluster"
  wait::for -t 30 -- kubectl get anvil ${resname} -n backend
  resource::check_count -n backend -r anvil -c 1
  resource::check -n backend anvil ${resname} -a "configmanagement.gke.io/managed=enabled"
  resource::check -n backend anvil ${resname} -l "app.kubernetes.io/managed-by=configmanagement.gke.io"

  debug::log "Modify custom resource in repo"
  oldresver=$(resource::resource_version anvil ${resname} -n backend)
  git::update "${YAML_DIR}/customresources/anvil-heavier.yaml" acme/namespaces/eng/backend/anvil.yaml
  git::commit

  debug::log "Checking that custom resource was updated on cluster"
  resource::wait_for_update -n backend -t 20 anvil "${resname}" "${oldresver}"
  wait::for -o "100" -t 30 -- \
    kubectl get anvil ${resname} -n backend \
      --output='jsonpath={.spec.lbs}'

  git::rm acme/namespaces/eng/backend/anvil.yaml
  git::commit

  debug::log "Checking that no custom resources exist on cluster"
  wait::for -t 30 -f -- kubectl get anvil ${resname} --all-namespaces
}

bats_test_function test_-24-7bFILE-5fNAME-7d-3a_CRD_deleted_before_repo_update
bats_test_function test_-24-7bFILE-5fNAME-7d-3a_Sync_and_update_structural_custom_resource
