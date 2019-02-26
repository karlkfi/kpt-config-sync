#!/bin/bash

set -u

load "../lib/debug"
load "../lib/git"
load "../lib/namespace"
load "../lib/policynode"
load "../lib/setup"
load "../lib/wait"

setup() {
  setup::common
  local active=()
  mapfile -t active < <(kubectl get ns -l "nomos.dev/testdata=true" | grep Active)
  if (( ${#active[@]} != 0 )); then
    local ns
    for ns in "${active[@]}"; do
      kubectl delete ns $ns
    done
  fi
  setup::git::initialize
  setup::git::init_acme
}

# This cleans up any namespaces that were created by a testcase
function teardown() {
  kubectl delete ns -l "nomos.dev/testdata=true" --ignore-not-found=true
  setup::common_teardown
}

@test "Namespace has enabled annotation and declared" {
  local ns=decl-namespace-annotation-enabled
  namespace::create $ns -a "config.gke.io/managed=enabled"
  namespace::declare $ns
  git::commit

  wait::for -t 10 -- namespace::check_exists $ns -l "config.gke.io/quota=true" -a "config.gke.io/managed=enabled"
  namespace::check_no_warning $ns
}

@test "Namespace exists and declared" {
  local ns=decl-namespace-annotation-none
  namespace::create $ns
  namespace::declare $ns
  git::commit

  namespace::check_exists $ns
  namespace::check_warning $ns
}

@test "Namespace does not exist and declared" {
  local ns=decl-namespace-noexist
  namespace::declare $ns
  git::commit

  namespace::check_exists $ns -l "config.gke.io/quota=true" -a "config.gke.io/managed=enabled"
  namespace::check_no_warning $ns
}

@test "Namespace has enabled annotation with no declarations" {
  local ns=undeclared-annotation-enabled
  namespace::create $ns -a "config.gke.io/managed=enabled"
  namespace::check_not_found $ns
}

@test "Namespace exists with no declaration" {
  local ns=undeclared-annotation-none
  namespace::create $ns
  namespace::check_warning $ns
}

@test "Namespace warn on invalid management annotation" {
  local ns=decl-invalid-annotation
  namespace::create $ns -a "nomos.dev/namespace-management=a-garbage-annotation"
  namespace::declare $ns
  git::commit

  wait::for -t 30 -- policynode::sync_token_eq "$ns" "$(git::hash)"
  selection=$(kubectl get pn ${ns} -ojson | jq -c ".status.syncState")
  [[ "${selection}" == "\"error\"" ]] || debug::error "policy node status should be error, not ${selection}"
  namespace::check_exists $ns -a "nomos.dev/namespace-management=a-garbage-annotation"
  namespace::check_warning $ns
}

@test "Sync labels and annotations from decls" {
  debug::log "Creating namespace"
  local ns=labels-and-annotations
  namespace::declare $ns \
    -l "foo-corp.com/awesome-controller-flavour=fuzzy" \
    -a "foo-corp.com/awesome-controller-mixin=green"
  git::commit

  debug::log "Checking namespace exists"
  namespace::check_exists $ns \
    -l "foo-corp.com/awesome-controller-flavour=fuzzy" \
    -a "foo-corp.com/awesome-controller-mixin=green"
}

