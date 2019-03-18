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

setup() {
  setup::common
  setup::git::initialize
  setup::git::init_acme
}

# This cleans up any CRDs that were created by a testcase
function teardown() {
  kubectl delete crd anvils.acme.com --ignore-not-found=true || true
  kubectl delete crd clusteranvils.acme.com --ignore-not-found=true || true
  wait::for -f -t 30 -- kubectl get crd anvils.acme.com
  wait::for -f -t 30 -- kubectl get crd clusteranvils.acme.com
  if [[ "$WATCH_PID" != "" ]]; then
    kill $WATCH_PID || true
    WATCH_PID=""
  fi
  setup::common_teardown
}

@test "CRD deleted before repo update" {
  local resname="e2e-test-anvil"
  kubectl apply -f "${YAML_DIR}/customresources/anvil-crd.yaml"
  kubectl apply -f "${YAML_DIR}/customresources/clusteranvil-crd.yaml"
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

@test "Sync custom namespace scoped resource" {
  local resname="e2e-test-anvil"

  debug::log "Adding CRDs out of band"
  kubectl apply -f "${YAML_DIR}/customresources/anvil-crd.yaml"
  kubectl apply -f "${YAML_DIR}/customresources/clusteranvil-crd.yaml"
  resource::check crd anvils.acme.com
  resource::check crd clusteranvils.acme.com

  debug::log "Updating repo with custom resource"
  git::add "${YAML_DIR}/customresources/anvil.yaml" acme/namespaces/eng/backend/anvil.yaml
  git::commit

  debug::log "Checking that custom resource appears on cluster"
  wait::for -t 30 -- kubectl get anvil ${resname} -n backend
  resource::check_count -n backend -r anvil -c 1
  resource::check -n backend anvil ${resname} -a "configmanagement.gke.io/managed=enabled"

  debug::log "Modify custom resource in repo"
  oldresver=$(resource::resource_version anvil ${resname} -n backend)
  git::update "${YAML_DIR}/customresources/anvil-heavier.yaml" acme/namespaces/eng/backend/anvil.yaml
  git::commit

  debug::log "Checking that custom resource was updated on cluster"
  resource::wait_for_update -n backend -t 20 anvil "${resname}" "${oldresver}"
  selection=$(kubectl get anvil ${resname} -n backend -ojson | jq -c ".spec.lbs")
  [[ "${selection}" == "100" ]] || debug::error "custom resource weight should be updated to 100, not ${selection}"

  debug::log "Moving custom resource to another namespace"
  git::rm acme/namespaces/eng/backend/anvil.yaml
  git::add "${YAML_DIR}/customresources/anvil.yaml" acme/namespaces/rnd/new-prj/anvil.yaml
  git::commit

  debug::log "Checking that custom resource moved to another namespace"
  wait::for -t 30 -- kubectl get anvil ${resname} -n new-prj
  wait::for -f -t 30 -- kubectl get anvil ${resname} -n backend
  resource::check_count -n new-prj -r anvil -c 1
  resource::check -n new-prj anvil ${resname} -a "configmanagement.gke.io/managed=enabled"

  debug::log "Add anvil-heavier with management disabled"
  kubectl apply -f "${YAML_DIR}/customresources/anvil-heavier-management-disabled.yaml" -n newer-prj
  wait::for -t 30 -- kubectl get anvil ${resname} -n newer-prj
  debug::log "Update repo with new resource"
  git::add "${YAML_DIR}/customresources/anvil.yaml" acme/namespaces/rnd/newer-prj/anvil.yaml
  debug::log "We don't expect this commit to result in a change on the cluster since management is disabled"
  git::commit

  debug::log "Verify that anvil-heavier (lbs: 100) is still on cluster, and not anvil (lbs: 10)"
  wait::for -t 30 -- namespaceconfig::sync_token_eq newer-prj "$(git::hash)"
  debug::log "Anvil events: $(kubectl get events | grep "Anvil")"
  debug::log "Ensure we didn't overwrite the management label"
  resource::check -n newer-prj anvil ${resname} -a "configmanagement.gke.io/managed=disabled"
  selection=$(kubectl get anvil ${resname} -n newer-prj -ojson | jq -c ".spec.lbs")
  [[ "${selection}" == "100" ]] || debug::error "unmanaged custom resource weight should be 100, not ${selection}"

  debug::log "Remove all custom resources from cluster"
  git::rm acme/namespaces/rnd/new-prj/anvil.yaml
  git::commit
  #kubectl delete anvil ${resname} -n newer-prj

  debug::log "Checking that no custom resources exist on cluster"
  wait::for -t 30 -f -- kubectl get anvil ${resname} --all-namespaces
}

@test "Sync custom cluster scoped resource" {
  local resname="e2e-test-clusteranvil"

  debug::log "Adding CRDs out of band"
  kubectl apply -f "${YAML_DIR}/customresources/anvil-crd.yaml"
  kubectl apply -f "${YAML_DIR}/customresources/clusteranvil-crd.yaml"
  resource::check crd anvils.acme.com
  resource::check crd clusteranvils.acme.com

  kubectl get clusteranvil -w -oyaml > clusteranvil.log &
  WATCH_PID=$!

  debug::log "Updating repo with custom cluster resource"
  git::add "${YAML_DIR}/customresources/clusteranvil.yaml" acme/cluster/clusteranvil.yaml
  git::commit

  debug::log "Checking that custom cluster resource appears on cluster"
  wait::for -t 30 -- kubectl get clusteranvil ${resname}
  resource::check_count -r clusteranvil -c 1
  resource::check clusteranvil ${resname} -a "configmanagement.gke.io/managed=enabled"
  selection=$(kubectl get clusteranvil ${resname} -ojson | jq -c ".spec.lbs")
  [[ "${selection}" == "10" ]] || debug::error "custom resource weight should be 10, not ${selection}"

  debug::log "Modify custom resource in repo"
  oldresver=$(resource::resource_version clusteranvil ${resname})
  git::update "${YAML_DIR}/customresources/clusteranvil-heavier.yaml" acme/cluster/clusteranvil.yaml
  git::commit

  debug::log "Checking that custom resource was updated on cluster"
  resource::wait_for_update -t 20 clusteranvil "${resname}" "${oldresver}"
  selection=$(kubectl get clusteranvil ${resname} -ojson | jq -c ".spec.lbs")
  if [[ "${selection}" != "100" ]]; then
    debug::log "Output from kubectl get clusteranvil -w -oyaml:\\n$(cat clusteranvil.log)"
    debug::error "custom resource weight should be updated to 100, not ${selection}"
  fi

  debug::log "Remove custom cluster resource from cluster"
  git::rm acme/cluster/clusteranvil.yaml
  git::commit

  debug::log "Checking that custom cluster resource removed from cluster"
  wait::for -t 30 -f -- kubectl get clusteranvil ${resname}

  debug::log "Add clusteranvil-heavier with management disabled"
  kubectl apply -f "${YAML_DIR}/customresources/clusteranvil-heavier-management-disabled.yaml"
  wait::for -t 30 -- kubectl get clusteranvil ${resname}
  debug::log "Update repo with new resource"
  git::add "${YAML_DIR}/customresources/clusteranvil.yaml" acme/cluster/clusteranvil.yaml
  debug::log "We don't expect this commit to result in a change on the cluster since management is disabled"
  git::commit

  debug::log "Verify that clusteranvil-heavier (lbs: 100) is still on cluster, and not clusteranvil (lbs: 10)"
  wait::for -t 30 -- namespaceconfig::sync_token_eq newer-prj "$(git::hash)"
  debug::log "ClusterAnvil events: $(kubectl get events | grep "ClusterAnvil")"
  debug::log "Ensure we didn't overwrite the management label"
  resource::check clusteranvil ${resname} -a "configmanagement.gke.io/managed=disabled"
  selection=$(kubectl get clusteranvil ${resname} -ojson | jq -c ".spec.lbs")
  [[ "${selection}" == "100" ]] || debug::error "unmanaged custom resource weight should be 100, not ${selection}"

  git::rm acme/cluster/clusteranvil.yaml
  git::commit
  #kubectl delete clusteranvil ${resname}

  debug::log "Checking that no custom resources exist on cluster"
  wait::for -t 30 -f -- kubectl get anvil ${resname} --all-namespaces
}
