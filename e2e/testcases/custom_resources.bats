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
  resource::check -n backend anvil ${resname} -l "app.kubernetes.io/managed-by=configmanagement.gke.io"

  debug::log "Ensure we added the Nomos-specific labels and annotations"
  annotationValue=$(kubectl get anvil ${resname} -n backend -ojson | jq '.metadata.annotations."configmanagement.gke.io/cluster-name"')
  [[ "${annotationValue}" != "null" ]] || debug::error "cluster name annotation not added"
  annotationValue=$(kubectl get anvil ${resname} -n backend -ojson | jq '.metadata.annotations."configmanagement.gke.io/source-path"')
  [[ "${annotationValue}" != "null" ]] || debug::error "source path annotation not added"

  debug::log "Modify custom resource in repo"
  oldresver=$(resource::resource_version anvil ${resname} -n backend)
  git::update "${YAML_DIR}/customresources/anvil-heavier.yaml" acme/namespaces/eng/backend/anvil.yaml
  git::commit

  debug::log "Checking that custom resource was updated on cluster"
  resource::wait_for_update -n backend -t 20 anvil "${resname}" "${oldresver}"
  wait::for -o "100" -t 30 -- \
    kubectl get anvil ${resname} -n backend \
      --output='jsonpath={.spec.lbs}'

  debug::log "Moving custom resource to another namespace"
  git::rm acme/namespaces/eng/backend/anvil.yaml
  git::add "${YAML_DIR}/customresources/anvil.yaml" acme/namespaces/rnd/new-prj/anvil.yaml
  git::commit

  debug::log "Checking that custom resource moved to another namespace"
  wait::for -t 30 -- kubectl get anvil ${resname} -n new-prj
  wait::for -f -t 30 -- kubectl get anvil ${resname} -n backend
  resource::check_count -n new-prj -r anvil -c 1
  resource::check -n new-prj anvil ${resname} -a "configmanagement.gke.io/managed=enabled"

  debug::log "Update repo with anvil-heavier-management-disabled"
  git::update "${YAML_DIR}/customresources/anvil-heavier-management-disabled.yaml" acme/namespaces/rnd/new-prj/anvil.yaml
  debug::log "We don't expect this commit to result in a change on the cluster since management is disabled"
  git::commit

  debug::log "Verify that anvil (lbs: 10) is still on cluster, and not heavier anvil (lbs: 100)"
  wait::for -t 30 -- namespaceconfig::sync_token_eq new-prj "$(git::hash)"
  debug::log "Anvil events: $(kubectl get events | grep "Anvil")"
  wait::for -o "10" -t 30 -- \
    kubectl get anvil ${resname} -n new-prj \
      --output='jsonpath={.spec.lbs}'

  debug::log "Ensure we removed the Nomos-specific labels and annotations"
  annotationValue=$(kubectl get -n new-prj anvil ${resname} -ojson | jq '.metadata.annotations."configmanagement.gke.io/managed"')
  [[ "${annotationValue}" == "null" ]] || debug::error "management annotation not removed"
  annotationValue=$(kubectl get -n new-prj anvil ${resname} -ojson | jq '.metadata.annotations."configmanagement.gke.io/source-path"')
  [[ "${annotationValue}" == "null" ]] || debug::error "source path annotation not removed"
  labelValue=$(kubectl get -n new-prj anvil ${resname} -ojson | jq '.metadata.annotations."configmanagement.gke.io/token"')
  [[ "${labelValue}" == "null" ]] || debug::error "token annotation not removed"
  labelValue=$(kubectl get -n new-prj anvil ${resname} -ojson | jq '.metadata.labels."app.kubernetes.io/managed-by"')
  [[ "${labelValue}" == "null" ]] || debug::error "managed-by annotation not removed"

  debug::log "Remove all custom resources from cluster"
  git::rm acme/namespaces/rnd/new-prj/anvil.yaml
  git::commit

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
  wait::for -o "10" -t 30 -- \
    kubectl get clusteranvil ${resname} \
      --output='jsonpath={.spec.lbs}'

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

  debug::log "Update clusteranvil"
  kubectl apply -f "${YAML_DIR}/customresources/clusteranvil.yaml"
  wait::for -t 30 -- kubectl get clusteranvil ${resname}

  debug::log "Update repo with clusteranvil-heavier-management-disabled"
  git::add "${YAML_DIR}/customresources/clusteranvil-heavier-management-disabled.yaml" acme/cluster/clusteranvil.yaml
  debug::log "We don't expect this commit to result in a change on the cluster since management is disabled"
  git::commit

  debug::log "Verify that clusteranvil (lbs: 10) is still on cluster, and not clusteranvil-heavier (lbs: 100)"
  debug::log "ClusterAnvil events: $(kubectl get events | grep "ClusterAnvil")"
  wait::for -o "10" -t 30 -- \
    kubectl get clusteranvil ${resname} \
      --output='jsonpath={.spec.lbs}'

  debug::log "Ensure we didn't add a management label"
  wait::for -f -t 5 -- resource::check -n new-prj anvil ${resname} -a "configmanagement.gke.io/managed=enabled"

  git::rm acme/cluster/clusteranvil.yaml
  git::commit

  debug::log "Checking that no custom resources exist on cluster"
  wait::for -t 30 -f -- kubectl get anvil ${resname} --all-namespaces
}
