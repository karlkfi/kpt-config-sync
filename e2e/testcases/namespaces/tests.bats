#!/bin/bash

set -u

load common

@test "namespace has full label and declared" {
  local ns=decl-namespace-label-full
  create_cluster_namespace $ns full
  create_namespace $ns
  git::commit

  check_ns_exists $ns full
  check_ns_no_warn $ns
}

@test "namespace exists and declared" {
  local ns=decl-namespace-label-none
  create_cluster_namespace $ns
  create_namespace $ns
  git::commit

  check_ns_exists $ns
  check_ns_warn $ns
}

@test "namespace does not exist and declared" {
  local ns=decl-namespace-noexist
  create_namespace $ns
  git::commit

  check_ns_exists $ns full
  check_ns_no_warn $ns
}

@test "namespace has policies label and declared" {
  local ns=decl-namespace-label-policies
  create_cluster_namespace $ns policies
  create_namespace $ns
  git::commit

  check_ns_exists $ns policies
  check_ns_no_warn $ns
}

@test "namespace has full label with no declarations" {
  local ns=undeclared-label-full
  create_cluster_namespace $ns full
  check_ns_not_exists $ns
}

@test "namespace exists with no declaration" {
  local ns=undeclared-label-none
  create_cluster_namespace $ns
  check_ns_warn $ns
}

@test "namespace has policies label with no declaration" {
  local ns=undeclared-label-policies
  create_cluster_namespace $ns policies
  check_ns_warn $ns
}

@test "namespace has full label and declared as policyspace" {
  local ns=decl-policyspace-label-full
  create_cluster_namespace $ns full
  create_policyspace $ns
  git::commit
  wait::for "kubectl get pn $ns"

  check_ns_not_exists $ns
}

@test "namespace exists and declared as policyspace" {
  local ns=decl-policyspace-label-none
  create_cluster_namespace $ns
  create_policyspace $ns
  git::commit

  wait::for "kubectl get pn $ns"

  check_ns_exists $ns
  check_ns_warn $ns
}

@test "namespace has policies label and declared as policyspace" {
  local ns=decl-policyspace-label-policies
  create_cluster_namespace $ns policies
  create_policyspace $ns
  git::commit

  wait::for "kubectl get pn $ns"

  check_ns_exists $ns policies
  check_ns_warn $ns
}

@test "namespace has full label and reserved" {
  local ns=decl-reserved-label-full
  create_cluster_namespace $ns full
  create_reserved $ns
  git::commit

  check_ns_exists $ns full
  check_ns_warn $ns
}

@test "namespace exists and reserved" {
  local ns=decl-reserved-label-none
  create_cluster_namespace $ns
  create_reserved $ns
  git::commit

  check_ns_exists $ns
  check_ns_no_warn $ns
}

@test "namespace does not exist and reserved" {
  local ns=decl-reserved-noexist
  create_reserved $ns
  git::commit

  check_ns_not_exists $ns full
  check_ns_no_warn $ns
}

@test "namespace has policies label and reserved" {
  local ns=decl-reserved-label-policies
  create_cluster_namespace $ns policies
  create_reserved $ns
  git::commit

  check_ns_exists $ns policies
  check_ns_warn $ns
}
