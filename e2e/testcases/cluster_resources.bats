#!/bin/bash

set -u

load "../lib/clusterrole"
load "../lib/debug"
load "../lib/git"
load "../lib/resource"
load "../lib/setup"
load "../lib/wait"

setup() {
  setup::common
  setup::git::initialize
  setup::git::init_acme
}

teardown() {
  setup::common_teardown
}

YAML_DIR=${BATS_TEST_DIRNAME}/../testdata

function sync_token_eq() {
  local expect="${1:-}"
  local stoken
  stoken="$(kubectl get clusterpolicy -ojsonpath='{.items[0].status.syncToken}')"
  if [[ "$stoken" != "$expect" ]]; then
    echo "syncToken is $stoken waiting for $expect"
    return 1
  fi
}

function check_cluster_scoped_resource() {
  local res="${1:-}"
  local jsonpath="${2:-}"
  local create="${3:-}"
  local modify="${4:-}"
  local resext="${5:-}"

  local resname=e2e-test-${res}
  local respath=acme/cluster/e2e-${res}.yaml

  [ -n "$res" ]
  [ -n "$jsonpath" ]
  [ -n "$create" ]
  [ -n "$modify" ]

  git::add ${YAML_DIR}/${res}${resext}-create.yaml ${respath}
  git::commit

  # wait for resource creation
  wait::for kubectl get ${res} ${resname}

  # check selection for correct value
  local selection
  selection=$(kubectl get ${res} ${resname} -ojson | jq -c "${jsonpath}")
  [[ "$selection" == "$create" ]] || debug::error "$selection != $create"

  local resver
  resver=$(resource::resource_version ${res} e2e-test-${res})
  git::update ${YAML_DIR}/${res}${resext}-modify.yaml ${respath}
  git::commit

  # wait for update
  resource::wait_for_update -t 20 ${res} ${resname} ${resver}
  selection=$(kubectl get ${res} ${resname} -ojson | jq -c "${jsonpath}")
  [[ "$selection" == "$modify" ]] || debug::error "$selection != $modify"

  # verify that importToken has been updated from the commit above
  local itoken="$(kubectl get clusterpolicy -ojsonpath='{.items[0].spec.importToken}')"
  git::check_hash "$itoken"

  # verify that syncToken has been updated as well
  wait::for -- sync_token_eq "$itoken"

  # delete item
  git::rm ${respath}
  git::commit

  # check removed
  wait::for -f -- kubectl get ${res} ${resname}
}

@test "Local kubectl apply to clusterroles gets reverted" {
  local clusterrole_name
  clusterrole_name="e2e-test-clusterrole"
  debug::log "Add initial clusterrole"
  git::add ${YAML_DIR}/clusterrole-modify.yaml acme/cluster/clusterrole.yaml
  git::commit -m "Adds an initial clusterrole"

  debug::log "Check initial state of the object"
  wait::for -o "[get list create]" -t 10 -- \
    kubectl get clusterroles "${clusterrole_name}" -o jsonpath='{.rules[0].verbs}'
  debug::log "Modify the clusterrole on the apiserver to remove 'create' verb from the role"
  kubectl apply --filename="${YAML_DIR}/clusterrole-create.yaml"
  debug::log "Nomos should revert the permission to what it is in the source of truth"
  wait::for -t 10 -- \
    clusterrole::validate_management_selection ${clusterrole_name} '.rules[0].verbs' '["get","list","create"]'
}

@test "Lifecycle for clusterroles" {
  check_cluster_scoped_resource \
    clusterrole \
    ".rules[].verbs" \
    '["get","list"]' \
    '["get","list","create"]'
}

@test "Lifecycle for clusterrolebindings" {
  check_cluster_scoped_resource \
    clusterrolebinding \
    ".subjects[].name" \
    '"cyril@acme.com"' \
    '"cheryl@acme.com"' \
    "-2"
}

@test "Lifecycle for podsecuritypolicies" {
  check_cluster_scoped_resource \
    podsecuritypolicy \
    ".spec.privileged" \
    'true' \
    'null'
}
