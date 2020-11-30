#!/bin/bash

set -euo pipefail

load "../lib/debug"
load "../lib/git"
load "../lib/resource"
load "../lib/setup"
load "../lib/wait"

YAML_DIR=${BATS_TEST_DIRNAME}/../testdata
WATCH_PID=""

FILE_NAME="$(basename "${BATS_TEST_FILENAME}" '.bats')"

# It isn't guaranteed that the caller has already launched and connected to a
# Kubernetes cluster when they parse this file, so we execute this from within
# tests.
get_minor_version() {
  kubectl version -ojson | jq -r '.serverVersion.minor[0:2]'
}

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

@test "${FILE_NAME}: CRD deleted before repo update" {
  local minor_version
  minor_version=$(get_minor_version)
  if (( minor_version < 16 )); then
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
  wait::for -t 60 -- nomos::repo_synced

  debug::log "Custom resource exists on cluster"
  kubectl get anvil ${resname} -n backend

  debug::log "Removing CRD for custom resource"
  kubectl delete crd anvils.acme.com --ignore-not-found=true || true

  debug::log "Removing anvil from repo"
  git::rm acme/namespaces/eng/backend/anvil.yaml
  git::commit
  wait::for -t 60 -- nomos::repo_synced

  debug::log "Waiting for sync removal"
  ! kubectl get sync anvils.acme.com
}
