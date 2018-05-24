#!/bin/bash

load ../../lib/assert
load ../../lib/git
load ../../lib/wait
load ../../lib/setup

# This cleans up any namespaces that were created by a testcase
function local_teardown() {
  echo kubectl delete ns -l "nomos.dev/testdata=true"
}

# Directly creates a namespace on the cluster with optional label for nomos management.
#
# Arguments
#   name: The namespace name
#   label: (optional) the nomos.dev/namespace-management label value
#
function create_cluster_namespace() {
  local name=$1
  local label=${2:-}
  local tmp=$BATS_TMPDIR/namespace-${name}.yaml
  if [[ "${label}" != "" ]]; then
    label="    nomos.dev/namespace-management: ${label}"
  fi
  cat > $tmp <<- EOM
apiVersion: v1
kind: Namespace
metadata:
  name: ${name}
  labels:
    nomos.dev/testdata: "true"
${label}
EOM
  echo "Creating Cluster Namespace:"
  cat ${tmp}
  echo
  kubectl apply -f ${tmp}
}

# Creates a policyspace in the git repo.
#
# Arguments
#   path: The path to the policyspace directory under acme with the last portion being the policyspace name.
#
function create_policyspace() {
  local path=$1
  local dst=acme/${path}/placeholder
  local abs_dst=${TEST_REPO}/${dst}
  mkdir -p $(dirname ${abs_dst})
  touch ${abs_dst}
  git -C ${TEST_REPO} add ${dst}
}

# Creates a namespace in the git repo.
#
# Arguments
#   path: The path to the namesapce directory under acme with the last portion being the namesapce name.
#
function create_namespace() {
  local path=$1
  local name=$(basename ${path})
  local dst=acme/${path}/namespace.yaml
  local abs_dst=${TEST_REPO}/${dst}

  mkdir -p $(dirname ${TEST_REPO}/${dst})
  cat > ${abs_dst} <<- EOM
apiVersion: v1
kind: Namespace
metadata:
  name: ${name}
EOM
  git -C ${TEST_REPO} add ${dst}
}

# Designates a reserved namespace in the git repo.
#
# Arguments
#   name: The name of the reserved namespace.
#
function create_reserved() {
  local name=$1
  local dst=acme/nomos-reserved-namespaces.yaml

  cat > ${TEST_REPO}/${dst} <<- EOM
apiVersion: v1
kind: ConfigMap
metadata:
  name: nomos-reserved-namespaces
data:
  ${name}: reserved
EOM
  git -C ${TEST_REPO} add ${dst}
}

# Checks that a namespace exists on the cluster.
#
# Arguments
#   name: The name of the namespace
#   label: (optional) the value for the nomos.dev/namespace-management label to check.
function check_ns_exists() {
  local ns=$1
  local label=${2:-}

  wait::for_success "kubectl get ns ${ns}"
  if [[ "${label}" != "" ]]; then
    run kubectl get ns ${ns} -ojsonpath="{.metadata.labels['nomos\.dev/namespace-management']}"
    assert::contains "${label}"
  fi
}

# Checks a namespace's parent label matches a specific value.
#
# Arguments
#   name: The name of the namespace
#   parent: The parent value to check
function check_ns_parent() {
  local ns=$1
  local parent=$2

  run kubectl get ns ${ns} -ojsonpath="{.metadata.labels['nomos\.dev/parent-name']}"
  assert::contains "${parent}"
}

# Checks that a namespace does not exist
#
# Arguments
#   name: The name of the namespace
function check_ns_not_exists() {
  local ns=$1
  wait::for_failure "kubectl get ns ${ns}"
  run kubectl get ns ${ns}
  assert::contains "NotFound"
}

# Checks that a namespace has a warning event.
#
# Arguments
#   name: The name of the namespace
function check_ns_warn() {
  local ns=$1
  # TODO: check for warning event or status
}

# Checks that a namespace has no warning event.
#
# Arguments
#   name: The name of the namespace
function check_ns_no_warn() {
  local ns=$1
  # TODO: check for no warning event or status
}
