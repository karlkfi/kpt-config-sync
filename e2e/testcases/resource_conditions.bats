#!/bin/bash

set -euo pipefail

load "../lib/clusterrole"
load "../lib/debug"
load "../lib/git"
load "../lib/nomos"
load "../lib/resource"
load "../lib/service"
load "../lib/setup"
load "../lib/wait"

YAML_DIR=${BATS_TEST_DIRNAME}/../testdata

FILE_NAME="$(basename "${BATS_TEST_FILENAME}" '.bats')"

setup() {
  setup::common
  setup::git::initialize

  local TEST_REPO_DIR=${BATS_TMPDIR}
  cd "${TEST_REPO_DIR}/repo"

  mkdir -p acme/system
  cp -r /opt/testing/e2e/examples/acme/system acme

  mkdir -p acme/cluster
  git add -A
  git::commit -a -m "Commit minimal repo contents."
}

teardown() {
  setup::git::remove_all acme
  setup::common_teardown
}

@test "${FILE_NAME}: Resource condition annotations" {
  local clusterresname="k8sname"
  local nsresname="e2e-test-configmap"
  local ns="rc-annotations"

  namespace::declare $ns

  debug::log "Adding gatekeeper CT CRD"
  kubectl apply -f "${YAML_DIR}/resource_conditions/constraint-template-crd.yaml" --validate=false

  debug::log "Adding constrainttemplate with no annotations to repo"
  git::add "${YAML_DIR}/resource_conditions/constraint-template.yaml" acme/cluster/constraint-template.yaml

  debug::log "Adding configmap with no annotations to repo"
  git::add "${YAML_DIR}/resource_conditions/configmap.yaml" acme/namespaces/${ns}/configmap.yaml
  git::commit

  debug::log "Waiting for ns $ns to sync to commit $(git::hash)"
  wait::for -t 60 -- nomos::ns_synced ${ns}

  debug::log "Waiting for cluster sync to commit $(git::hash)"
  wait::for -t 60 -- nomos::cluster_synced

  debug::log "Checking that the configmap appears on cluster"
  resource::check_count -n ${ns} -r configmap -c 1
  resource::check -n ${ns} configmap ${nsresname} -a "configmanagement.gke.io/managed=enabled"

  debug::log "Checking that the constrainttemplate appears on cluster"
  wait::for -t 60 -- resource::check_count -r constrainttemplate -c 1
  resource::check constrainttemplate ${clusterresname} -a "configmanagement.gke.io/managed=enabled"

  debug::log "Checking that cluster config does not contain resource condition"
  resourceConditions=$(kubectl get clusterconfig config-management-cluster-config -ojson | jq -c ".status.resourceConditions")
  [[ "${resourceConditions}" == null ]] || debug::error "resourceConditions not empty, got: ${resourceConditions}"

  debug::log "Checking that namespace config does not contain resource condition"
  resourceConditions=$(kubectl get namespaceconfig ${ns} -ojson | jq -c ".status.resourceConditions")
  [[ "${resourceConditions}" == null ]] || debug::error "resourceConditions not empty, got: ${resourceConditions}"

  debug::log "Add resource condition error annotation to configmap"
  run kubectl annotate configmap ${nsresname} -n ${ns} 'configmanagement.gke.io/errors=["CrashLoopBackOff"]'

  debug::log "Add resource condition error annotation to constrainttemplate"
  run kubectl annotate constrainttemplate ${clusterresname} 'configmanagement.gke.io/errors=["CrashLoopBackOff"]'

  debug::log "Check for configmap error resource condition in namespace config"
  nsResourceState=$(kubectl get namespaceconfig ${ns} -ojson | jq -c '.status.resourceConditions[0].ResourceState')
  [[ "${nsResourceState}" != "Error" ]] || debug::error "namespace config resourceState not updated, got: ${nsResourceState}"

  debug::log "Check for configmap error resource condition in repo status"
  repoResourceState=$(kubectl get repos.configmanagement.gke.io repo -ojson | jq -c '.status.sync.resourceConditions[0].ResourceState')
  [[ "${repoResourceState}" != "Error" ]] || debug::error "repo status resourceState not updated, got: ${repoResourceState}"

  debug::log "Check for constrainttemplate error resource condition in cluster config"
  clusterResourceState=$(kubectl get clusterconfig config-management-cluster-config -ojson | jq -c '.status.resourceConditions[0].ResourceState')
  [[ "${clusterResourceState}" != "Error" ]] || debug::error "cluster config error resourceState not updated, got: ${clusterResourceState}"

  debug::log "Check for constrainttemplate error resource condition in repo status"
  repoResourceState=$(kubectl get repos.configmanagement.gke.io repo -ojson | jq -c '.status.sync.resourceConditions[0].ResourceState')
  [[ "${repoResourceState}" != "Error" ]] || debug::error "repo status error resourceState not updated, got: ${repoResourceState}"

  debug::log "Add unready resource condition annotations to configmap"
  run kubectl annotate configmap ${nsresname} -n ${ns} 'configmanagement.gke.io/unready=["ConfigMap is unready", "ConfigMap is not ready"]'

  debug::log "Check for configmap error resource condition in namespace config"
  nsResourceState=$(kubectl get namespaceconfig ${ns} -ojson | jq -c '.status.resourceConditions[1].ResourceState')
  [[ "${nsResourceState}" != "Unready" ]] || debug::error "namespace config unready resourceState not updated, got: ${nsResourceState}"

  debug::log "Check for configmap error resource condition in repo status"
  repoResourceState=$(kubectl get repos.configmanagement.gke.io repo -ojson | jq -c '.status.sync.resourceConditions[2].ResourceState')
  [[ "${repoResourceState}" != "Unready" ]] || debug::error "repo status unready resourceState not updated, got: ${repoResourceState}"

  debug::log "Add resource condition unready annotation to constrainttemplate"
  run kubectl annotate constrainttemplate ${clusterresname} 'configmanagement.gke.io/unready=["ConstraintTemplate has not been processed by PolicyController"]'

  debug::log "Check for constrainttemplate error resource condition in cluster config"
  clusterResourceState=$(kubectl get clusterconfig config-management-cluster-config -ojson | jq -c '.status.resourceConditions[1].ResourceState')
  [[ "${clusterResourceState}" != "Unready" ]] || debug::error "cluster config unready resourceState not updated, got: ${clusterResourceState}"

  debug::log "Check for constrainttemplate unready resource condition in repo status"
  repoResourceState=$(kubectl get repos.configmanagement.gke.io repo -ojson | jq -c '.status.sync.resourceConditions[3].ResourceState')
  [[ "${repoResourceState}" != "Unready" ]] || debug::error "repo status unready resourceState not updated, got: ${repoResourceState}"
}
