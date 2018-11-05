#!/bin/bash

set -euo pipefail

YAML_DIR=${BATS_TEST_DIRNAME}/../testdata

load ../lib/loader

# This cleans up any CRDs that were created by a testcase
function local_teardown() {
  # TODO(119278434): Remove git stuff when importer is fixed.
  git::rm acme/system/acme-sync.yaml
  git::commit
  kubectl delete crd anvils.acme.com --ignore-not-found=true || true
  kubectl delete crd clusteranvils.acme.com --ignore-not-found=true || true
}

@test "Sync custom namespace scoped resource" {
  local resname="e2e-test-anvil"

  debug::log "Adding CRDs out of band"
  kubectl apply -f "${YAML_DIR}/customresources/anvil-crd.yaml"
  kubectl apply -f "${YAML_DIR}/customresources/clusteranvil-crd.yaml"
  resource::check crd anvils.acme.com
  resource::check crd clusteranvils.acme.com

  debug::log "Updating repo with sync and custom resource"
  git::add "${YAML_DIR}/customresources/acme-sync.yaml" acme/system/acme-sync.yaml
  git::add "${YAML_DIR}/customresources/anvil.yaml" acme/namespaces/eng/backend/anvil.yaml
  git::commit

  debug::log "Checking that custom resource appears on cluster"
  wait::for -t 30 -- kubectl get anvil ${resname} -n backend
  resource::check_count -n backend -r anvil -c 1
  resource::check -n backend anvil ${resname} -l "nomos.dev/managed=full"

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
  resource::check_count -n new-prj -r anvil -c 1
  resource::check -n new-prj anvil ${resname} -l "nomos.dev/managed=full"

  debug::log "Add custom resource without managed label to custom resource and update repo with the same resource"
  kubectl apply -f "${YAML_DIR}/customresources/anvil-heavier.yaml" -n newer-prj
  wait::for -t 30 -- kubectl get anvil ${resname} -n newer-prj
  git::add "${YAML_DIR}/customresources/anvil.yaml" acme/namespaces/rnd/newer-prj/anvil.yaml
  git::commit

  debug::log "Checking that original resource remains on cluster"
  wait::event -t 30 UnmanagedResource -n newer-prj
  selection=$(kubectl get anvil ${resname} -n newer-prj -ojson | jq -c ".spec.lbs")
  [[ "${selection}" == "100" ]] || debug::error "unmanaged custom resource weight should be 100, not ${selection}"

  debug::log "Remove all custom resource from cluster"
  git::rm acme/namespaces/rnd/new-prj/anvil.yaml
  git::commit
  kubectl delete anvil ${resname} -n newer-prj

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

  debug::log "Updating repo with sync and custom cluster resource"
  git::add "${YAML_DIR}/customresources/acme-sync.yaml" acme/system/acme-sync.yaml
  git::add "${YAML_DIR}/customresources/clusteranvil.yaml" acme/cluster/clusteranvil.yaml
  git::commit

  debug::log "Checking that custom cluster resource appears on cluster"
  wait::for -t 30 -- kubectl get clusteranvil ${resname}
  resource::check_count -r clusteranvil -c 1
  resource::check clusteranvil ${resname} -l "nomos.dev/managed=full"
  selection=$(kubectl get clusteranvil ${resname} -ojson | jq -c ".spec.lbs")
  [[ "${selection}" == "10" ]] || debug::error "custom resource weight should be 10, not ${selection}"

  debug::log "Modify custom resource in repo"
  oldresver=$(resource::resource_version clusteranvil ${resname})
  git::update "${YAML_DIR}/customresources/clusteranvil-heavier.yaml" acme/cluster/clusteranvil.yaml
  git::commit

  debug::log "Checking that custom resource was updated on cluster"
  resource::wait_for_update -t 20 clusteranvil "${resname}" "${oldresver}"
  selection=$(kubectl get clusteranvil ${resname} -ojson | jq -c ".spec.lbs")
  [[ "${selection}" == "100" ]] || debug::error "custom resource weight should be updated to 100, not ${selection}"

  debug::log "Remove custom cluster resource from cluster"
  git::rm acme/cluster/clusteranvil.yaml
  git::commit

  debug::log "Checking that custom cluster resource removed from cluster"
  wait::for -t 30 -f -- kubectl get clusteranvil ${resname}
}
