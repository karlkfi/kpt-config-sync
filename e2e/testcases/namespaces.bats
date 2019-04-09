#!/bin/bash

set -u

load "../lib/debug"
load "../lib/git"
load "../lib/namespace"
load "../lib/namespaceconfig"
load "../lib/setup"
load "../lib/wait"

setup() {
  setup::common
  local active=()
  mapfile -t active < <(kubectl get ns -l "configmanagement.gke.io/testdata=true" | grep Active)
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
  kubectl delete ns -l "configmanagement.gke.io/testdata=true" --ignore-not-found=true
  setup::common_teardown
}

@test "Namespace has enabled annotation and declared" {
  local ns=decl-namespace-annotation-enabled
  namespace::create $ns -a "configmanagement.gke.io/managed=enabled"
  namespace::declare $ns
  git::commit

  wait::for -t 10 -- namespace::check_exists $ns -l "configmanagement.gke.io/quota=true" -a "configmanagement.gke.io/managed=enabled"
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

  namespace::check_exists $ns -l "configmanagement.gke.io/quota=true" -a "configmanagement.gke.io/managed=enabled"
  namespace::check_no_warning $ns
}

@test "Namespace has enabled annotation with no declarations" {
  local ns=undeclared-annotation-enabled
  namespace::create $ns -a "configmanagement.gke.io/managed=enabled"
  namespace::check_not_found $ns
}

@test "Namespace exists with no declaration" {
  local ns=undeclared-annotation-none
  namespace::create $ns
  namespace::check_warning $ns
}

@test "Namespace with management disabled gets cleaned" {
  local ns=unmanage-test

  debug::log "Declare namespace and wait for it to exist"
  namespace::declare $ns
  git::commit
  wait::for -t 30 -- namespace::check_exists $ns -a "configmanagement.gke.io/managed=enabled"

  debug::log "Ensure we added the Nomos-specific annotations and quota label"
  annotationValue=$(kubectl get ns $ns -ojson | jq '.metadata.annotations."configmanagement.gke.io/cluster-name"')
  [[ "${annotationValue}" != "null" ]] || debug::error "cluster name annotation not added"
  annotationValue=$(kubectl get ns $ns -ojson | jq '.metadata.annotations."configmanagement.gke.io/managed"')
  [[ "${annotationValue}" != "null" ]] || debug::error "management annotation not added"
  annotationValue=$(kubectl get ns $ns -ojson | jq '.metadata.annotations."configmanagement.gke.io/source-path"')
  [[ "${annotationValue}" != "null" ]] || debug::error "source path annotation not added"
  annotationValue=$(kubectl get ns $ns -ojson | jq '.metadata.annotations."configmanagement.gke.io/token"')
  [[ "${annotationValue}" != "null" ]] || debug::error "sync token annotation not added"
  annotationValue=$(kubectl get ns $ns -ojson | jq '.metadata.labels."configmanagement.gke.io/quota"')
  [[ "${annotationValue}" != "null" ]] || debug::error "quota annotation not added"

  debug::log "Declare management disabled on Namespace and wait for management to be disabled"
  namespace::declare $ns -a "configmanagement.gke.io/managed=disabled"
  git::commit
  wait::for -f -t 30 -- namespace::check_exists $ns -a "configmanagement.gke.io/managed=enabled"

  debug::log "Ensure we removed the Nomos-specific annotations"
  namespace::check_exists $ns
  annotationValue=$(kubectl get ns $ns -ojson | jq '.metadata.annotations."configmanagement.gke.io/cluster-name"')
  [[ "${annotationValue}" == "null" ]] || debug::error "cluster name annotation not removed"
  annotationValue=$(kubectl get ns $ns -ojson | jq '.metadata.annotations."configmanagement.gke.io/managed"')
  [[ "${annotationValue}" == "null" ]] || debug::error "management annotation not removed"
  annotationValue=$(kubectl get ns $ns -ojson | jq '.metadata.annotations."configmanagement.gke.io/source-path"')
  [[ "${annotationValue}" == "null" ]] || debug::error "source path annotation not removed"
  annotationValue=$(kubectl get ns $ns -ojson | jq '.metadata.annotations."configmanagement.gke.io/token"')
  [[ "${annotationValue}" == "null" ]] || debug::error "sync token annotation not removed"
  annotationValue=$(kubectl get ns $ns -ojson | jq '.metadata.labels."configmanagement.gke.io/quota"')
  [[ "${annotationValue}" == "null" ]] || debug::error "quota annotation not removed"
}

@test "Namespace with invalid management annotation cleaned" {
  local validns=valid-annotation
  local invalidns=invalid-annotation

  namespace::declare $validns
  namespace::create $invalidns -a "configmanagement.gke.io/managed=a-garbage-annotation"
  git::commit

  wait::for -t 30 -- namespace::check_exists $validns -a "configmanagement.gke.io/managed=enabled"
  namespace::check_exists $invalidns
  managedValue=$(kubectl get ns $invalidns -ojson | jq '.metadata.annotations."configmanagement.gke.io/managed"')
  [[ "${managedValue}" == "null" ]] || debug::error "invalid management annotation not removed"

  namespace::check_warning $invalidns
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
