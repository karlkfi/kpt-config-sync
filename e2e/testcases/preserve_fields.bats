#!/bin/bash

set -euo pipefail

load "../lib/clusterrole"
load "../lib/debug"
load "../lib/git"
load "../lib/resource"
load "../lib/service"
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
  mkdir -p acme/namespaces/backend
  namespace::declare backend
  git add -A
  git::commit -a -m "Commit minimal repo contents."
}

teardown() {
  setup::common_teardown
}

@test "Preserve autogenerated fields when syncing services" {
  local resname="e2e-test-service"
  local ns="backend"

  debug::log "Adding service to repo"
  git::add "${YAML_DIR}/preservefields/service.yaml" "acme/namespaces/${ns}/service.yaml"
  git::commit

  debug::log "Checking that the service appears on cluster"
  wait::for -t 60 -- kubectl get services ${resname} -n ${ns}
  resource::check_count -n ${ns} -r services -c 1
  resource::check -n ${ns} service ${resname} -a "configmanagement.gke.io/managed=enabled"

  debug::log "Checking that specified port is set"
  selector=".spec.ports[0].targetPort"
  targetPort=$(kubectl get service ${resname} -n ${ns} -ojson | jq -c "${selector}")
  [[ "${targetPort}" == "9376" ]] || debug::error "${selector} was not synced, got ${targetPort}"

  debug::log "Checking that autogenerated fields are preserved"
  wait::for -t 30 -- service::has_cluster_ip ${resname} ${ns}
  clusterIP=$(kubectl get service ${resname} -n ${ns} -ojson | jq -c ".spec.clusterIP")
  [[ "${clusterIP}" != "" ]] || debug::error "service.spec.clusterIP should not have been removed"

  debug::log "Modify custom resource in repo"
  oldresver=$(resource::resource_version services ${resname} -n ${ns})
  git::update "${YAML_DIR}/preservefields/service-modify.yaml" acme/namespaces/${ns}/service.yaml
  git::commit

  debug::log "Checking that autogenerated field is preserved after syncing an update"
  resource::wait_for_update -n ${ns} -t 30 services "${resname}" "${oldresver}"
  currentClusterIP=$(kubectl get service ${resname} -n ${ns} -ojson | jq -c ".spec.clusterIP")
  [[ "${currentClusterIP}" == "${clusterIP}" ]] || debug::error "service.spec.clusterIP shouldn't have changed from ${clusterIP} to ${currentClusterIP}"

  debug::log "Checking that specified port is updated"
  selector=".spec.ports[0].targetPort"
  targetPort=$(kubectl get service ${resname} -n ${ns} -ojson | jq -c "${selector}")
  [[ "${targetPort}" == "8080" ]] || debug::error "${selector} was not synced, got ${targetPort}"
}

@test "Preserve autogenerated fields when syncing ClusterRoles with aggregation" {
  local resname="aggregate"
  local expectRulesCount=2

  debug::log "Adding a ClusterRole that aggregates other ClusterRoles"
  git::add "${YAML_DIR}/preservefields/namespace-viewer-clusterrole.yaml" acme/cluster/namespace-viewer-clusterrole.yaml
  git::add "${YAML_DIR}/preservefields/rbac-viewer-clusterrole.yaml" acme/cluster/rbac-viewer-clusterrole.yaml
  git::add "${YAML_DIR}/preservefields/aggregate-clusterrole.yaml" acme/cluster/aggregate-clusterrole.yaml
  git::commit

  debug::log "Checking that the aggregate ClusterRole appears on cluster"
  wait::for -t 30 -- kubectl get clusterroles ${resname}
  resource::check clusterrole ${resname} -a "configmanagement.gke.io/managed=enabled"

  debug::log "Checking that autogenerated fields are preserved"
  wait::for -t 30 -- clusterrole::rules_count_eq ${resname} ${expectRulesCount}

  debug::log "Modify custom resource in repo"
  oldresver=$(resource::resource_version clusterrole ${resname})
  git::update "${YAML_DIR}/preservefields/aggregate-clusterrole-modify.yaml" acme/cluster/aggregate-clusterrole.yaml
  git::commit

  debug::log "Checking that autogenerated field is preserved after syncing an update"
  resource::wait_for_update -t 30 clusterrole "${resname}" "${oldresver}"
  rulesCount=$(kubectl get clusterrole ${resname} -ojson | jq -c ".rules | length")
  (( "${rulesCount}" == "${expectRulesCount}" )) || debug::error "clusterrole.rules count shouldn't have changed from ${expectRulesCount} to ${rulesCount}"
}

@test "Add/Update/Delete labels on configmap" {
  local resname="e2e-test-configmap"
  local ns="backend"

  debug::log "Adding configmap with no labels to repo"
  git::add "${YAML_DIR}/preservefields/configmap-no-labels-annotations.yaml" "acme/namespaces/${ns}/configmap.yaml"
  git::commit

  debug::log "Checking that the configmap with no labels appears on cluster"
  wait::for -t 30 -- kubectl get configmap ${resname} -n ${ns}
  resource::check_count -n ${ns} -r configmap -c 1
  resource::check -n ${ns} configmap ${resname} -a "configmanagement.gke.io/managed=enabled"

  debug::log "Checking that no user labels specified"
  wait::for -t 30 -- validate_configmap_labels ${resname} ${ns} '{"app.kubernetes.io/managed-by":"configmanagement.gke.io"}'

  debug::log "Add labels to configmap in repo"
  oldresver=$(resource::resource_version configmap ${resname} -n ${ns})
  git::update "${YAML_DIR}/preservefields/configmap-add-label.yaml" "acme/namespaces/${ns}/configmap.yaml"
  git::commit

  debug::log "Checking that label is added after syncing an update"
  resource::wait_for_update -n ${ns} -t 30 configmaps "${resname}" "${oldresver}"
  wait::for -t 30 -- validate_configmap_labels ${resname} ${ns} '{"app.kubernetes.io/managed-by":"configmanagement.gke.io","foo":"bar"}'

  debug::log "Update label for configmap in repo"
  oldresver=$(resource::resource_version configmap ${resname} -n ${ns})
  git::update "${YAML_DIR}/preservefields/configmap-update-label.yaml" "acme/namespaces/${ns}/configmap.yaml"
  git::commit

  debug::log "Checking that label is updated after syncing an update"
  resource::wait_for_update -n ${ns} -t 30 configmaps "${resname}" "${oldresver}"
  wait::for -t 30 -- validate_configmap_labels ${resname} ${ns} '{"app.kubernetes.io/managed-by":"configmanagement.gke.io","baz":"qux"}'

  debug::log "Delete label for configmap in repo"
  oldresver=$(resource::resource_version configmap ${resname} -n ${ns})
  git::update "${YAML_DIR}/preservefields/configmap-no-labels-annotations.yaml" "acme/namespaces/${ns}/configmap.yaml"
  git::commit

  debug::log "Checking that label is updated after syncing an update"
  resource::wait_for_update -n ${ns} -t 30 configmaps "${resname}" "${oldresver}"
  wait::for -t 30 -- validate_configmap_labels ${resname} ${ns} '{"app.kubernetes.io/managed-by":"configmanagement.gke.io"}'
}

function validate_configmap_labels() {
  local resname="${1:-}"
  local ns="${2:-}"
  local expected="${3:-}"
  local selector='.metadata.labels'
  local labels
  labels=$(kubectl get configmap "${resname}" -n "${ns}" -ojson | jq -c "${selector}")
  [[ "${labels}" == "${expected}" ]] || debug::error "${selector} was not synced, got: ${labels}"

  local config
  config=$(kubectl get configmap "${resname}" -n "${ns}" -ojson | jq -c '.metadata.annotations."kubectl.kubernetes.io/last-applied-configuration"' | sed  's/^.\(.*\).$/\1/' | xargs -0 printf '%b')
  labels=$(echo "${config}" | jq -c "${selector}")
  [[ "${labels}" == "${expected}" ]] || debug::error "last-applied-config ${selector} was not synced, got: ${labels}
  want: ${expected}"
}

@test "Add/Update/Delete annotations on configmap" {
  local resname="e2e-test-configmap"
  local ns="backend"

  debug::log "Adding configmap with no annotations to repo"
  git::add "${YAML_DIR}/preservefields/configmap-no-labels-annotations.yaml" acme/namespaces/${ns}/configmap.yaml
  git::commit

  debug::log "Checking that the configmap with no annotations appears on cluster"
  wait::for -t 30 -- kubectl get configmap ${resname} -n ${ns}
  resource::check_count -n ${ns} -r configmap -c 1
  resource::check -n ${ns} configmap ${resname} -a "configmanagement.gke.io/managed=enabled"

  debug::log "Checking that no annotations specified"
  wait::for -t 30 -- validate_configmap_annotations ${resname} ${ns}

  debug::log "Add annotations to configmap in repo"
  oldresver=$(resource::resource_version configmap ${resname} -n ${ns})
  git::update "${YAML_DIR}/preservefields/configmap-add-annotation.yaml" "acme/namespaces/${ns}/configmap.yaml"
  git::commit

  debug::log "Checking that annotation is added after syncing an update"
  resource::wait_for_update -n ${ns} -t 30 configmaps "${resname}" "${oldresver}"
  wait::for -t 30 -- validate_configmap_annotations ${resname} ${ns} ',"zoo"'

  debug::log "Update annotation for configmap in repo"
  oldresver=$(resource::resource_version configmap ${resname} -n ${ns})
  git::update "${YAML_DIR}/preservefields/configmap-update-annotation.yaml" "acme/namespaces/${ns}/configmap.yaml"
  git::commit

  debug::log "Checking that annotation is updated after syncing an update"
  resource::wait_for_update -n ${ns} -t 30 configmaps "${resname}" "${oldresver}"
  wait::for -t 30 -- validate_configmap_annotations ${resname} ${ns} ',"zaz"'

  debug::log "Delete annotation for configmap in repo"
  oldresver=$(resource::resource_version configmap ${resname} -n ${ns})
  git::update "${YAML_DIR}/preservefields/configmap-no-labels-annotations.yaml" "acme/namespaces/${ns}/configmap.yaml"
  git::commit

  debug::log "Checking that annotation is updated after syncing an update"
  resource::wait_for_update -n ${ns} -t 30 configmaps "${resname}" "${oldresver}"
  wait::for -t 30 -- validate_configmap_annotations ${resname} ${ns}
}

function validate_configmap_annotations() {
  local resname="${1:-}"
  local ns="${2:-}"
  local additional_keys="${3:-}"
  local selector='.metadata.annotations | keys | map(select(. != "configmanagement.gke.io/cluster-name")) | sort'
  local nested_default_keys='["configmanagement.gke.io/managed","configmanagement.gke.io/source-path","configmanagement.gke.io/token"'
  local default_keys
  default_keys="${nested_default_keys},\"kubectl.kubernetes.io/last-applied-configuration\""
  local annotations
  annotations=$(kubectl get configmap "${resname}" -n "${ns}" -ojson | jq -c "${selector}")
  [[ "${annotations}" == "${default_keys}${additional_keys}]" ]] || debug::error "${selector} for keys '${default_keys}${additional_keys}' was not synced, got: ${annotations}"

  local config
  config=$(kubectl get configmap "${resname}" -n "${ns}" -ojson | jq -c '.metadata.annotations."kubectl.kubernetes.io/last-applied-configuration"' | sed  's/^.\(.*\).$/\1/' | xargs -0 printf '%b')
  annotations=$(echo "${config}" | jq -c "${selector}")
  [[ "${annotations}" == "${nested_default_keys}${additional_keys}]" ]] || debug::error "last-applied-config ${selector}  was not synced, got ${annotations}"
}
