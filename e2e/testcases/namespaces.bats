#!/bin/bash

set -u

load ../lib/loader

# This cleans up any namespaces that were created by a testcase
function local_teardown() {
  kubectl delete ns -l "nomos.dev/testdata=true" --ignore-not-found=true || true
}

@test "Namespace has full label and declared" {
  local ns=decl-namespace-label-full
  namespace::create $ns -l "nomos.dev/namespace-management=full"
  namespace::declare $ns
  git::commit

  namespace::check_exists $ns -l "nomos.dev/namespace-management=full"
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

  namespace::check_exists $ns -l "nomos.dev/namespace-management=full"
  namespace::check_no_warning $ns
}

@test "Namespace has policies label and declared" {
  local ns=decl-namespace-label-policies
  namespace::create $ns -l "nomos.dev/namespace-management=policies"
  namespace::declare $ns
  git::commit

  namespace::check_exists $ns -l "nomos.dev/namespace-management=policies"
  namespace::check_no_warning $ns
}

@test "Namespace has full label with no declarations" {
  local ns=undeclared-label-full
  namespace::create $ns -l "nomos.dev/namespace-management=full"
  namespace::check_not_found $ns
}

@test "Namespace exists with no declaration" {
  local ns=undeclared-label-none
  namespace::create $ns
  namespace::check_warning $ns
}

@test "Namespace has policies label with no declaration" {
  local ns=undeclared-label-policies
  namespace::create $ns -l "nomos.dev/namespace-management=policies"
  namespace::check_warning $ns
}

@test "Namespace has full label and declared as policyspace" {
  local ns=decl-policyspace-label-full
  namespace::create $ns -l "nomos.dev/namespace-management=full"
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

@test "Namespace has policies label and declared as policyspace" {
  local ns=decl-policyspace-label-policies
  namespace::create $ns -l "nomos.dev/namespace-management=policies"
  namespace::declare_policyspace $ns
  git::commit

  wait::for kubectl get pn "$ns"

  namespace::check_exists $ns -l "nomos.dev/namespace-management=policies"
  namespace::check_warning $ns
}

@test "Namespace has full label and reserved" {
  local ns=decl-reserved-label-full
  namespace::create $ns -l "nomos.dev/namespace-management=full"
  namespace::declare_reserved $ns
  git::commit

  namespace::check_exists $ns -l "nomos.dev/namespace-management=full"
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

  namespace::check_not_found $ns full
  namespace::check_no_warning $ns
}

@test "Namespace has policies label and reserved" {
  local ns=decl-reserved-label-policies
  namespace::create ${ns} -l "nomos.dev/namespace-management=policies"
  namespace::declare_reserved $ns
  git::commit

  namespace::check_exists $ns -l "nomos.dev/namespace-management=policies"
  namespace::check_warning $ns
}

@test "Namespace warn on invalid management label" {
  local ns=decl-invalid-label
  namespace::create $ns -l "nomos.dev/namespace-management=a-garbage-label"
  namespace::declare $ns
  git::commit

  wait::event \
    -n default \
    InvalidManagementLabel
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

