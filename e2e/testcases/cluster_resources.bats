#!/bin/bash

set -u

load ../lib/loader

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

@test "Lifecycle for clusterroles" {
  check_cluster_scoped_resource \
    clusterrole \
    ".rules[].verbs" \
    '["get","list"]' \
    '["get","list","create"]'
}

@test "Lifecycle for clusterrolebindings 1" {
  skip # b/111606994
  check_cluster_scoped_resource \
    clusterrolebinding \
    ".roleRef.name" \
    '"edit"' \
    '"view"'
}

@test "Lifecycle for clusterrolebindings 2" {
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

