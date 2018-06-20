#!/bin/bash

set -u

load ../../lib/loader

# This cleans up any namespaces that were created by a testcase
function local_teardown() {
  kubectl delete ns -l "nomos.dev/testdata=true" --ignore-not-found=true || true
}

@test "Namespace has full label and declared" {
  local ns=decl-namespace-label-full
  namespace::create $ns full
  namespace::declare $ns
  git::commit

  namespace::check_exists $ns full
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

  namespace::check_exists $ns full
  namespace::check_no_warning $ns
}

@test "Namespace has policies label and declared" {
  local ns=decl-namespace-label-policies
  namespace::create $ns policies
  namespace::declare $ns
  git::commit

  namespace::check_exists $ns policies
  namespace::check_no_warning $ns
}

@test "Namespace has full label with no declarations" {
  local ns=undeclared-label-full
  namespace::create $ns full
  namespace::check_not_found $ns
}

@test "Namespace exists with no declaration" {
  local ns=undeclared-label-none
  namespace::create $ns
  namespace::check_warning $ns
}

@test "Namespace has policies label with no declaration" {
  local ns=undeclared-label-policies
  namespace::create $ns policies
  namespace::check_warning $ns
}

@test "Namespace has full label and declared as policyspace" {
  local ns=decl-policyspace-label-full
  namespace::create $ns full
  namespace::declare_policyspace $ns
  git::commit
  wait::for "kubectl get pn $ns"

  namespace::check_not_found $ns
}

@test "Namespace exists and declared as policyspace" {
  local ns=decl-policyspace-label-none
  namespace::create $ns
  namespace::declare_policyspace $ns
  git::commit

  wait::for "kubectl get pn $ns"

  namespace::check_exists $ns
  namespace::check_warning $ns
}

@test "Namespace has policies label and declared as policyspace" {
  local ns=decl-policyspace-label-policies
  namespace::create $ns policies
  namespace::declare_policyspace $ns
  git::commit

  wait::for "kubectl get pn $ns"

  namespace::check_exists $ns policies
  namespace::check_warning $ns
}

@test "Namespace has full label and reserved" {
  local ns=decl-reserved-label-full
  namespace::create $ns full
  namespace::declare_reserved $ns
  git::commit

  namespace::check_exists $ns full
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
  namespace::create $ns policies
  namespace::declare_reserved $ns
  git::commit

  namespace::check_exists $ns policies
  namespace::check_warning $ns
}

@test "namespace declared, has policies label, update parent" {
  local nsp=move-decl-policies
  local nsf=move-decl-full
  namespace::create ${nsp} policies
  namespace::declare_policyspace src
  namespace::declare_policyspace dst
  namespace::declare src/$nsp
  namespace::declare src/$nsf
  git::commit

  namespace::check_exists $nsp policies
  namespace::check_exists $nsf full

  local nsp_ver=$(resource::resource_version ns ${nsp})
  local nsf_ver=$(resource::resource_version ns ${nsf})

  namespace::declare dst/$nsp
  namespace::declare dst/$nsf
  git::rm acme/src/$nsp/namespace.yaml
  git::rm acme/src/$nsf/namespace.yaml
  git::commit

  namespace::check_exists $nsp policies
  namespace::check_exists $nsf full

  resource::wait_for_update ns ${nsp} ${nsp_ver}
  resource::wait_for_update ns ${nsf} ${nsf_ver}

  namespace::check_parent $nsp dst
  namespace::check_parent $nsf dst
}
