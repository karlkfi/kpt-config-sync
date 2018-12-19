#!/bin/bash

set -u

load "../lib/debug"
load "../lib/git"
load "../lib/namespace"
load "../lib/policynode"
load "../lib/setup"
load "../lib/wait"

# This cleans up any namespaces that were created by a testcase
function local_teardown() {
  kubectl delete ns -l "nomos.dev/testdata=true" --ignore-not-found=true || true
}

@test "Namespace has enabled label and declared" {
  local ns=decl-namespace-label-enabled
  namespace::create $ns -l "nomos.dev/managed=enabled"
  namespace::declare $ns
  git::commit

  namespace::check_exists $ns -l "nomos.dev/managed=enabled"
  namespace::check_no_warning $ns
}

@test "Namespace exists and declared" {
  local ns=decl-namespace-label-none
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

  namespace::check_exists $ns -l "nomos.dev/managed=enabled"
  namespace::check_no_warning $ns
}

@test "Namespace has enabled label with no declarations" {
  local ns=undeclared-label-enabled
  namespace::create $ns -l "nomos.dev/managed=enabled"
  namespace::check_not_found $ns
}

@test "Namespace exists with no declaration" {
  local ns=undeclared-label-none
  namespace::create $ns
  namespace::check_warning $ns
}

@test "Namespace has enabled label and declared as policyspace" {
  local ns=decl-policyspace-label-enabled
  namespace::create $ns -l "nomos.dev/managed=enabled"
  namespace::declare_policyspace $ns
  git::commit
  wait::for kubectl get pn "$ns"

  namespace::check_not_found $ns
}

@test "Namespace exists and declared as policyspace" {
  local ns=decl-policyspace-label-none
  namespace::create $ns
  namespace::declare_policyspace $ns
  git::commit

  wait::for kubectl get pn "$ns"

  namespace::check_exists $ns
  namespace::check_warning $ns
}

@test "Namespace has enabled label and reserved" {
  local ns=decl-reserved-label-enabled
  namespace::create $ns -l "nomos.dev/managed=enabled"
  namespace::declare_reserved $ns
  git::commit

  namespace::check_exists $ns -l "nomos.dev/managed=enabled"
  namespace::check_warning $ns
}

@test "Namespace exists and reserved" {
  local ns=decl-reserved-label-none
  namespace::create $ns
  namespace::declare_reserved $ns
  git::commit

  namespace::check_exists $ns
  namespace::check_no_warning $ns
}

@test "Namespace does not exist and reserved" {
  local ns=decl-reserved-noexist
  namespace::declare_reserved $ns
  git::commit

  namespace::check_not_found $ns enabled
  namespace::check_no_warning $ns
}

@test "Namespace warn on invalid management label" {
  local ns=decl-invalid-label
  namespace::create $ns -l "nomos.dev/namespace-management=a-garbage-label"
  namespace::declare $ns
  git::commit

  wait::for -t 30 -- policynode::sync_token_eq "$ns" "$(git::hash)"
  selection=$(kubectl get pn ${ns} -ojson | jq -c ".status.syncState")
  [[ "${selection}" == "\"error\"" ]] || debug::error "policy node status should be error, not ${selection}"
  namespace::check_exists $ns -l "nomos.dev/namespace-management=a-garbage-label"
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

