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

test_setup() {
  setup::git::initialize
  setup::git::commit_minimal_repo_contents --skip_wait
  mkdir -p acme/cluster
}

test_teardown() {
  setup::git::remove_all acme
}

@test "${FILE_NAME}: Preserve autogenerated fields when syncing services" {
  local resname="e2e-test-service"
  local ns="autogen-fields"
  namespace::declare $ns -l "testdata=true"

  debug::log "Adding service to repo"
  git::add "${YAML_DIR}/preservefields/service.yaml" "acme/namespaces/${ns}/service.yaml"
  git::commit
  wait::for -t 60 -- nomos::repo_synced

  debug::log "Checking that the service appears on cluster"
  kubectl get services ${resname} -n ${ns}
  resource::check_count -n ${ns} -r services -c 1
  resource::check -n ${ns} service ${resname} -a "configmanagement.gke.io/managed=enabled"

  debug::log "Checking that specified port is set"
  selector=".spec.ports[0].targetPort"
  targetPort=$(kubectl get service ${resname} -n ${ns} -ojson | jq -c "${selector}")
  [[ "${targetPort}" == "9376" ]] || debug::error "${selector} was not synced, got ${targetPort}"

  # If strategic merge is NOT being used, Nomos and Kubernetes fight over
  # nodePort.  Nomos constantly deletes the value, and Kubernetes assigns a
  # random value each time.
  # 5 seconds is more than enough time for this to happen.
  debug::log "Checking that strategic merge is used"
  selector=".spec.ports[0].nodePort"
  nodePort1=$(kubectl get service ${resname} -n ${ns} -ojson | jq -c "${selector}")
  sleep 5
  nodePort2=$(kubectl get service ${resname} -n ${ns} -ojson | jq -c "${selector}")
  [[ "${nodePort1}" == "${nodePort2}" ]] || debug::error "not using strategic merge patch"

  debug::log "Checking that autogenerated fields are preserved"
  wait::for -t 60 -- service::has_cluster_ip ${resname} ${ns}
  clusterIP=$(kubectl get service ${resname} -n ${ns} -ojson | jq -c ".spec.clusterIP")
  [[ "${clusterIP}" != "" ]] || debug::error "service.spec.clusterIP should not have been removed"

  debug::log "Modify Service"
  git::update "${YAML_DIR}/preservefields/service-modify.yaml" acme/namespaces/${ns}/service.yaml
  git::commit
  wait::for -t 60 -- nomos::repo_synced

  debug::log "Checking that autogenerated field is preserved after syncing an update"
  currentClusterIP=$(kubectl get service ${resname} -n ${ns} -ojson | jq -c ".spec.clusterIP")
  [[ "${currentClusterIP}" == "${clusterIP}" ]] || debug::error "service.spec.clusterIP shouldn't have changed from ${clusterIP} to ${currentClusterIP}"

  debug::log "Checking that specified port is updated"
  selector=".spec.ports[0].targetPort"
  targetPort=$(kubectl get service ${resname} -n ${ns} -ojson | jq -c "${selector}")
  [[ "${targetPort}" == "8080" ]] || debug::error "${selector} was not synced, got ${targetPort}"
}

@test "${FILE_NAME}: Preserve autogenerated fields when syncing ClusterRoles with aggregation" {
  local resname="aggregate"
  local expectRulesCount=2

  debug::log "Adding a ClusterRole that aggregates other ClusterRoles"
  git::add "${YAML_DIR}/preservefields/namespace-viewer-clusterrole.yaml" acme/cluster/namespace-viewer-clusterrole.yaml
  git::add "${YAML_DIR}/preservefields/rbac-viewer-clusterrole.yaml" acme/cluster/rbac-viewer-clusterrole.yaml
  git::add "${YAML_DIR}/preservefields/aggregate-clusterrole.yaml" acme/cluster/aggregate-clusterrole.yaml
  git::commit

  debug::log "Waiting for sync to complete"
  wait::for -t 60 -- nomos::repo_synced

  debug::log "Checking that the aggregate ClusterRole appears on cluster"
  resource::check clusterrole ${resname} -a "configmanagement.gke.io/managed=enabled"

  debug::log "Checking that autogenerated fields are preserved"
  wait::for -t 60 -- clusterrole::rules_count_eq ${resname} ${expectRulesCount}
  debug::log "resource=$(kubectl get clusterrole ${resname} -ojson)"

  debug::log "Modify custom resource in repo"
  git::update "${YAML_DIR}/preservefields/aggregate-clusterrole-modify.yaml" acme/cluster/aggregate-clusterrole.yaml
  git::commit
  debug::log "resource=$(kubectl get clusterrole ${resname} -ojson)"

  debug::log "Checking that autogenerated field is preserved after syncing an update"
  wait::for -t 60 -- nomos::repo_synced
  local resource
  resource=$(kubectl get clusterrole ${resname} -ojson)
  rulesCount=$(jq -c ".rules | length" <<< "$resource")
  (( "${rulesCount}" == "${expectRulesCount}" )) || debug::error "clusterrole.rules count shouldn't have changed from ${expectRulesCount} to ${rulesCount} resource=${resource}"
}

# TODO(b/160032776): remove this test once all users are past 1.4.0
@test "${FILE_NAME}: Preserve and migrate contents of old last-applied-configuration annotation" {
  debug::log "Adding a ClusterRole"
  git::add "${YAML_DIR}/preservefields/namespace-viewer-clusterrole.yaml" acme/cluster/namespace-viewer-clusterrole.yaml
  git::commit

  debug::log "Waiting for sync to complete"
  wait::for -t 60 -- nomos::repo_synced

  debug::log "Checking that the ClusterRole appears on cluster"
  resource::check clusterrole namespace-viewer -a "configmanagement.gke.io/managed=enabled"

  debug::log "Replacing ClusterRole with old annotation version"
  kubectl replace -f "${YAML_DIR}/preservefields/namespace-viewer-clusterrole-replace.yaml"

  debug::log "Waiting for ClusterRole annotations to be migrated"
  validate_annotation_migrated namespace-viewer
}

function validate_annotation_migrated() {
  local resname="${1:-}"
  local selector='.metadata.annotations | keys | map(select(. != "configmanagement.gke.io/cluster-name")) | sort'
  local nested_default_keys='"configmanagement.gke.io/managed","configmanagement.gke.io/source-path","configmanagement.gke.io/token"'
  local default_keys
  default_keys="\"configmanagement.gke.io/declared-config\",${nested_default_keys}"
  # This first check verifies that all of the annotation keys that we expect to be present are in fact present. See validate_configmap_annotations for full details.
  local annotations
  annotations=$(kubectl get clusterrole "${resname}" -ojson | jq -c "${selector}")
  [[ "${annotations}" == "[${default_keys}]" ]] || debug::error "${selector} for keys '${default_keys}' was not updated, got: ${annotations}"

  # This second check verifies that the content of the "declared-config" annotation contains all of the annotation keys that we expect to be present.
  local config
  config=$(kubectl get clusterrole "${resname}" -ojson | jq -c '.metadata.annotations."configmanagement.gke.io/declared-config"')
  config=$(unquote_and_unescape "${config}")
  annotations=$(echo "${config}" | jq -c "${selector}")
  [[ "${annotations}" == "[${nested_default_keys}]" ]] || debug::error "declared-config ${selector} was not updated, got ${annotations}"
}

@test "${FILE_NAME}: Add/Update/Delete labels" {
  local resname="e2e-test-configmap"
  local ns="crud-labels"
  namespace::declare $ns -l "testdata=true"

  debug::log "Adding configmap with no labels to repo"
  git::add "${YAML_DIR}/preservefields/configmap-no-labels-annotations.yaml" "acme/namespaces/${ns}/configmap.yaml"
  git::commit
  wait::for -t 60 -- nomos::repo_synced

  debug::log "Checking that the configmap with no labels appears on cluster"
  wait::for -t 60 -- kubectl get configmap ${resname} -n ${ns}
  resource::check_count -n ${ns} -r configmap -c 1 -l "app.kubernetes.io/managed-by=configmanagement.gke.io"
  resource::check -n ${ns} configmap ${resname} -a "configmanagement.gke.io/managed=enabled"

  debug::log "Checking that no user labels specified"
  validate_configmap_labels ${resname} ${ns} '{"app.kubernetes.io/managed-by":"configmanagement.gke.io"}'

  debug::log "Add labels to configmap in repo"
  git::update "${YAML_DIR}/preservefields/configmap-add-label.yaml" "acme/namespaces/${ns}/configmap.yaml"
  git::commit
  wait::for -t 60 -- nomos::repo_synced

  debug::log "Checking that label is added after syncing an update"
  validate_configmap_labels ${resname} ${ns} '{"app.kubernetes.io/managed-by":"configmanagement.gke.io","foo":"bar"}'

  debug::log "Update label for configmap in repo"
  git::update "${YAML_DIR}/preservefields/configmap-update-label.yaml" "acme/namespaces/${ns}/configmap.yaml"
  git::commit
  wait::for -t 60 -- nomos::repo_synced

  debug::log "Checking that label is updated after syncing an update"
  validate_configmap_labels ${resname} ${ns} '{"app.kubernetes.io/managed-by":"configmanagement.gke.io","baz":"qux"}'

  debug::log "Delete label for configmap in repo"
  git::update "${YAML_DIR}/preservefields/configmap-no-labels-annotations.yaml" "acme/namespaces/${ns}/configmap.yaml"
  git::commit
  wait::for -t 60 -- nomos::repo_synced

  debug::log "Checking that label is updated after syncing an update"
  validate_configmap_labels ${resname} ${ns} '{"app.kubernetes.io/managed-by":"configmanagement.gke.io"}'
}

function validate_configmap_labels() {
  local resname="${1:-}"
  local ns="${2:-}"
  local expected="${3:-}"
  local selector='.metadata.labels'
  # This first check just verifies that all of the label keys that we expect to be present are in fact present. See validate_configmap_annotations for full details.
  local labels
  labels=$(kubectl get configmap "${resname}" -n "${ns}" -ojson | jq -c "${selector}")
  [[ "${labels}" == "${expected}" ]] || debug::error "${selector} was not synced, got: ${labels}"

  # This second check verifies that the content of the "declared-config" annotation contains all of the label keys that we expect to be present.
  local config
  config=$(kubectl get configmap "${resname}" -n "${ns}" -ojson | jq -c '.metadata.annotations."configmanagement.gke.io/declared-config"')
  config=$(unquote_and_unescape "${config}")
  labels=$(echo "${config}" | jq -c "${selector}")
  [[ "${labels}" == "${expected}" ]] || debug::error "declared-config ${selector} was not synced, got: ${labels}
  want: ${expected}"
}

@test "${FILE_NAME}: Add/Update/Delete annotations" {
  local resname="e2e-test-configmap"
  local ns="crud-annotations"
  namespace::declare $ns -l "testdata=true"

  debug::log "Adding configmap with no annotations to repo"
  git::add "${YAML_DIR}/preservefields/configmap-no-labels-annotations.yaml" acme/namespaces/${ns}/configmap.yaml
  git::commit

  debug::log "Waiting for ns $ns to sync to commit $(git::hash)"
  wait::for -t 60 -- nomos::repo_synced

  debug::log "Checking that the configmap with no annotations appears on cluster"
  resource::check_count -n ${ns} -r configmap -c 1 -l "app.kubernetes.io/managed-by=configmanagement.gke.io"
  resource::check -n ${ns} configmap ${resname} -a "configmanagement.gke.io/managed=enabled"

  debug::log "Checking that no annotations specified"
  validate_configmap_annotations ${resname} ${ns}

  debug::log "Add annotations to configmap in repo"
  git::update "${YAML_DIR}/preservefields/configmap-add-annotation.yaml" "acme/namespaces/${ns}/configmap.yaml"
  git::commit

  debug::log "Waiting for ns $ns to sync to commit"
  wait::for -t 60 -- nomos::repo_synced

  debug::log "Checking that annotation is added after syncing an update"
  validate_configmap_annotations ${resname} ${ns} ',"zoo"'

  debug::log "Update annotation for configmap in repo"
  git::update "${YAML_DIR}/preservefields/configmap-update-annotation.yaml" "acme/namespaces/${ns}/configmap.yaml"
  git::commit

  debug::log "Waiting for ns $ns to sync to commit"
  wait::for -t 60 -- nomos::repo_synced

  debug::log "Checking that annotation is updated after syncing an update"
  validate_configmap_annotations ${resname} ${ns} ',"zaz"'

  debug::log "Delete annotation for configmap in repo"
  git::update "${YAML_DIR}/preservefields/configmap-no-labels-annotations.yaml" "acme/namespaces/${ns}/configmap.yaml"
  git::commit

  debug::log "Waiting for ns $ns to sync to commit"
  wait::for -t 60 -- nomos::repo_synced

  debug::log "Checking that annotation is updated after syncing an update"
  validate_configmap_annotations ${resname} ${ns}
}

function unquote_and_unescape() {
  tmp="$1"
  # Use prefix/suffix commands to unquote the value.
  tmp="${tmp%\"}"
  tmp="${tmp#\"}"
  # Use printf to unescape value.
  # Disable shellcheck since '%b' and '%s' don't seem to work properly.
  # shellcheck disable=SC2059
  printf "${tmp}"
}

function validate_configmap_annotations() {
  local resname="${1:-}"
  local ns="${2:-}"
  local additional_keys="${3:-}"
  # This selector will pull all of the keys out of the annotations map, remove the cluster-name annotation, and sort them.
  local selector='.metadata.annotations | keys | map(select(. != "configmanagement.gke.io/cluster-name")) | sort'
  # These are annotation keys that we expect to be nested inside the content of the "declared-config" annotation.
  local nested_default_keys='"configmanagement.gke.io/managed","configmanagement.gke.io/source-path","configmanagement.gke.io/token"'
  # These are annotation keys that we expect to be present in the annotations map of the resource.
  local default_keys
  default_keys="\"configmanagement.gke.io/declared-config\",${nested_default_keys}"
  # We initialize annotations by applying the selector above to the json of the resource.
  local annotations
  annotations=$(kubectl get configmap "${resname}" -n "${ns}" -ojson | jq -c "${selector}")
  # We verify that all annotation keys are present in the resource's annotations map.
  [[ "${annotations}" == "[${default_keys}${additional_keys}]" ]] || debug::error "${selector} for keys '${default_keys}${additional_keys}' was not synced, got: ${annotations}"

  # We initialize config by reading the content of the "declared-config" annotation and formatting it nicely for jq.
  local config
  config=$(kubectl get configmap "${resname}" -n "${ns}" -ojson | jq -c '.metadata.annotations."configmanagement.gke.io/declared-config"')
  config=$(unquote_and_unescape "${config}")
  # We reinitialize annotations by applying the same selector above to the nested content we pulled from the "declared-config" annotation.
  annotations=$(echo "${config}" | jq -c "${selector}")
  # We verify that all nested annotation keys are present in the resource's annotations map as defined in "declared-config".
  [[ "${annotations}" == "[${nested_default_keys}${additional_keys}]" ]] || debug::error "declared-config ${selector}  was not synced, got ${annotations}"
}
