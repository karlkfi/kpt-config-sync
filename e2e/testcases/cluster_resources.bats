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
  local TEST_REPO_DIR=${BATS_TMPDIR}
  cd "${TEST_REPO_DIR}/repo"

  mkdir -p acme/system
  cp -r /opt/testing/e2e/examples/acme/system acme

  mkdir -p acme/cluster
  git add -A
  git::commit -a -m "Commit minimal repo contents."
}

teardown() {
  setup::common_teardown
}

YAML_DIR=${BATS_TEST_DIRNAME}/../testdata

function sync_token_eq() {
  local expect="${1:-}"
  local stoken
  stoken="$(kubectl get clusterconfig -ojsonpath='{.items[0].status.token}')"
  if [[ "$stoken" != "$expect" ]]; then
    echo "sync token is $stoken waiting for $expect"
    return 1
  fi
}

function resource_field_eq() {
  local res="${1:-}"
  local resname="${2:-}"
  local jsonpath="${3:-}"
  local expect="${4:-}"

  # wait for resource creation
  wait::for -t 20 -- kubectl get ${res} ${resname}

  # check selection for correct value
  local selection
  selection=$(kubectl get ${res} ${resname} -ojson | jq -c "${jsonpath}")
  [[ "$selection" == "$expect" ]] || debug::error "$selection != $expect"
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

  # check selection for correct value
  wait::for -t 30 -- resource_field_eq ${res} ${resname} ${jsonpath} ${create}

  local resver
  resver=$(resource::resource_version ${res} e2e-test-${res})
  git::update ${YAML_DIR}/${res}${resext}-modify.yaml ${respath}
  git::commit

  # wait for update
  resource::wait_for_update -t 20 ${res} ${resname} ${resver}
  selection=$(kubectl get ${res} ${resname} -ojson | jq -c "${jsonpath}")
  [[ "$selection" == "$modify" ]] || debug::error "$selection != $modify"

  # verify that import token has been updated from the commit above
  local itoken="$(kubectl get clusterconfig -ojsonpath='{.items[0].spec.token}')"
  git::check_hash "$itoken"

  # verify that sync token has been updated as well
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
