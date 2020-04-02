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
  local clusterresname="e2e-test-clusterrole"
  local nsresname="e2e-test-configmap"
  local ns="rc-annotations"

  namespace::declare $ns

  debug::log "Adding clusterrole with no annotations to repo"
  git::add "${YAML_DIR}/resource_conditions/clusterrole.yaml" acme/cluster/clusterrole.yaml

  debug::log "Adding configmap with no annotations to repo"
  git::add "${YAML_DIR}/resource_conditions/configmap.yaml" acme/namespaces/${ns}/configmap.yaml
  git::commit

  debug::log "Waiting for ns $ns to sync to commit $(git::hash)"
  wait::for -t 60 -- nomos::ns_synced ${ns}

  debug::log "Waiting for cluster sync to commit $(git::hash)"
  wait::for -t 60 -- nomos::cluster_synced

  debug::log "Checking that the configmap appears on cluster"
  resource::check -n ${ns} configmap ${nsresname} -a "configmanagement.gke.io/managed=enabled"

  debug::log "Checking that the clusterrole appears on cluster"
  resource::check clusterrole ${clusterresname} -a "configmanagement.gke.io/managed=enabled"

  debug::log "Checking that cluster config does not contain resource condition"
  resourceConditions=$(kubectl get clusterconfig config-management-cluster-config -ojson | jq -c ".status.resourceConditions")
  [[ "${resourceConditions}" == null ]] || debug::error "resourceConditions not empty, got: ${resourceConditions}"

  debug::log "Checking that namespace config does not contain resource condition"
  resourceConditions=$(kubectl get namespaceconfig ${ns} -ojson | jq -c ".status.resourceConditions")
  [[ "${resourceConditions}" == null ]] || debug::error "resourceConditions not empty, got: ${resourceConditions}"

  # Test adding error annotations

  debug::log "Add resource condition error annotation to configmap"
  run kubectl annotate configmap ${nsresname} -n ${ns} 'configmanagement.gke.io/errors=["CrashLoopBackOff"]'

  debug::log "Add resource condition error annotation to clusterrole"
  run kubectl annotate clusterrole ${clusterresname} 'configmanagement.gke.io/errors=["CrashLoopBackOff"]'

  debug::log "Check for configmap error resource condition in namespace config"
  wait::for -t 10 -- test '"Error"' == "$(kubectl get namespaceconfig ${ns} -ojson | jq -c '.status.resourceConditions[0].resourceState')"

  debug::log "Check for configmap error resource condition in repo status"
  wait::for -t 10 -- test '"Error"' == "$(kubectl get repos.configmanagement.gke.io repo -ojson | jq -c '.status.sync.resourceConditions[0].resourceState')"

  debug::log "Check for clusterrole error resource condition in cluster config"
  wait::for -t 10 -- test '"Error"' == "$(kubectl get clusterconfig config-management-cluster-config -ojson | jq -c '.status.resourceConditions[0].resourceState')"

  debug::log "Check for clusterrole error resource condition in repo status"
  wait::for -t 10 -- test '"Error"' == "$(kubectl get repos.configmanagement.gke.io repo -ojson | jq -c '.status.sync.resourceConditions[1].resourceState')"

  # Test removing error annotations

  debug::log "Remove resource condition annotations from configmap"
  run kubectl annotate configmap ${nsresname} -n ${ns} 'configmanagement.gke.io/errors-'

  debug::log "Remove resource condition annotations from clusterrole"
  run kubectl annotate clusterrole ${clusterresname} 'configmanagement.gke.io/errors-'

  debug::log "Check that namespace config does not contain resource conditions"
  wait::for -t 10 -- test null == "$(kubectl get namespaceconfig ${ns} -ojson | jq -c ".status.resourceConditions")"

  debug::log "Check that cluster config does not contain resource conditions"
  wait::for -t 10 -- test null == "$(kubectl get clusterconfig config-management-cluster-config -ojson | jq -c ".status.resourceConditions")"

  debug::log "Check that repo does not contain resource conditions"
  wait::for -t 10 -- test null == "$(kubectl get repos.configmanagement.gke.io repo -ojson | jq -c '.status.sync.resourceConditions')"

  # Test adding reconciling annotations

  debug::log "Add reconciling resource condition annotations to configmap"
  run kubectl annotate configmap ${nsresname} -n ${ns} 'configmanagement.gke.io/reconciling=["ConfigMap is incomplete", "ConfigMap is not ready"]'

  debug::log "Add resource condition reconciling annotation to clusterrole"
  run kubectl annotate clusterrole ${clusterresname} 'configmanagement.gke.io/reconciling=["ClusterRole needs... something..."]'

  debug::log "Check for configmap reconciling resource condition in namespace config"
  wait::for -t 10 -- test '"Reconciling"' == "$(kubectl get namespaceconfig ${ns} -ojson | jq -c '.status.resourceConditions[0].resourceState')"

  debug::log "Check for configmap reconciling resource condition in repo status"
  wait::for -t 10 -- test '"Reconciling"' == "$(kubectl get repos.configmanagement.gke.io repo -ojson | jq -c '.status.sync.resourceConditions[0].resourceState')"

  debug::log "Check for clusterrole reconciling resource condition in cluster config"
  wait::for -t 10 -- test '"Reconciling"' == "$(kubectl get clusterconfig config-management-cluster-config -ojson | jq -c '.status.resourceConditions[0].resourceState')"

  debug::log "Check for clusterrole reconciling resource condition in repo status"
  wait::for -t 10 -- test '"Reconciling"' == "$(kubectl get repos.configmanagement.gke.io repo -ojson | jq -c '.status.sync.resourceConditions[1].resourceState')"

  # Test removing reconciling annotations

  debug::log "Remove resource condition annotations from configmap"
  run kubectl annotate configmap ${nsresname} -n ${ns} 'configmanagement.gke.io/reconciling-'

  debug::log "Remove resource condition annotations from clusterrole"
  run kubectl annotate clusterrole ${clusterresname} 'configmanagement.gke.io/reconciling-'

  debug::log "Check that namespace config does not contain resource conditions"
  wait::for -t 10 -- test null == "$(kubectl get namespaceconfig ${ns} -ojson | jq -c ".status.resourceConditions")"

  debug::log "Check that cluster config does not contain resource conditions"
  wait::for -t 10 -- test null == "$(kubectl get clusterconfig config-management-cluster-config -ojson | jq -c ".status.resourceConditions")"

  debug::log "Check that repo does not contain resource conditions"
  wait::for -t 10 -- test null == "$(kubectl get repos.configmanagement.gke.io repo -ojson | jq -c '.status.sync.resourceConditions')"
}

@test "${FILE_NAME}: constraint template gets status annotations" {
  local resname="k8sname"

  debug::log "Adding gatekeeper CT CRD"
  kubectl apply -f "${YAML_DIR}/resource_conditions/constraint-template-crd.yaml" --validate=false

  git::add "${YAML_DIR}/resource_conditions/constraint-template.yaml" acme/cluster/constraint-template.yaml
  git::commit

  debug::log "Waiting for cluster sync to commit $(git::hash)"
  wait::for -t 60 -- nomos::cluster_synced

  debug::log "Waiting for CT to get reconciling annotation"
  wait::for -t 60 -- resource::check constrainttemplate ${resname} -a 'configmanagement.gke.io/reconciling=[\"ConstraintTemplate has not been created\"]'

  kubectl delete crd constrainttemplates.templates.gatekeeper.sh
}

@test "${FILE_NAME}: constraint gets status annotations" {
  local resname="prod-pod-is-fun"

  debug::log "Adding gatekeeper constraint CRD"
  kubectl apply -f "${YAML_DIR}/resource_conditions/constraint-crd.yaml" --validate=false

  git::add "${YAML_DIR}/resource_conditions/constraint.yaml" acme/cluster/constraint.yaml
  git::commit

  debug::log "Waiting for cluster sync to commit $(git::hash)"
  wait::for -t 60 -- nomos::cluster_synced

  debug::log "Waiting for constraint to get reconciling annotation"
  wait::for -t 120 -- resource::check funpods ${resname} -a 'configmanagement.gke.io/reconciling=[\"Constraint has not been processed by PolicyController\"]'

  kubectl delete crd funpods.constraints.gatekeeper.sh
}
