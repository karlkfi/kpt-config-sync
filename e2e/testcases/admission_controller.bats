#!/bin/bash

set -euo pipefail

YAML_DIR=${BATS_TEST_DIRNAME}/../testdata

load "../lib/git"
load "../lib/configmap"
load "../lib/setup"
load "../lib/wait"

FILE_NAME="$(basename "${BATS_TEST_FILENAME}" '.bats')"

# TEST_LABEL is used to tag test objects so we can clean them up during teardown
TEST_LABEL="configmanagement.gke.io/testdata=true"

# CHILD_DIRS defines all the child namespaces in the acme repo. Used for convenience
CHILD_DIRS=(
# eng/
    analytics
    backend
    frontend
# rnd/
    newer-prj
    new-prj
)

setup() {
  setup::common
  setup::git::initialize

  # init acme without the example quotas
  setup::git::init_without acme -- acme/namespaces/eng/quota.yaml acme/namespaces/rnd/quota.yaml acme/namespaces/eng/backend/quota.yaml

  # ensure admissions controller is installed
  wait::for kubectl get deployment -n config-management-system resourcequota-admission-controller
  wait::for kubectl get validatingwebhookconfigurations resource-quota.configmanagement.gke.io
}

teardown() {
  setup::common_teardown

  # Cleanup test specific resources not created by Nomos (i.e. not in the git repo)
  # the kubectl binary in the e2e docker img does not seem to accept the --all-namespaces flag,
  # so we work around it by looping. It might be the kubectl version (1.13).
  for child_dir in "${CHILD_DIRS[@]}"; do
    wait::for kubectl delete configmaps,pods -n="${child_dir}" -l "${TEST_LABEL}" --ignore-not-found
  done
}

@test "${FILE_NAME}: Enforce quota at one level below abstract namespace" {
  debug::log "Adding ResourceQuota to abstract namespace 'eng' with 'configmaps: 2'"
  git::add "${YAML_DIR}"/resourcequotas/resourcequota-configs.yaml acme/namespaces/eng/rq-configmaps.yaml
  git::commit

  debug::log "Verifying ResourceQuota installed in child namespaces of 'eng'"
  for child_dir in "${CHILD_DIRS[@]:0:3}"; do
    wait::for -s -t 15 -o "2 0" -- \
      kubectl get resourcequota config-management-resource-quota -n="${child_dir}"  \
        -o=jsonpath="{.status['.hard.configmaps', '.used.configmaps']}"
  done

  debug::log "Creating 2 dummy configmaps, each in a different child namespace"
  for child_dir in "${CHILD_DIRS[@]:0:2}"; do
    wait::for -s -t 15 -- \
      kubectl create configmap "${child_dir}" --from-literal=cat.fur=fluffy -n="${child_dir}" \
       && kubectl label configmap "${child_dir}" "${TEST_LABEL}" -n="${child_dir}"
  done

  debug::log "Attempting to add 3rd configmap, which should exceed the quota of 2"
  wait::for -f -t 15 -- \
    kubectl create configmap "${CHILD_DIRS[2]}" --from-literal=cat.fur=fluffy -n="${CHILD_DIRS[2]}"
}

@test "${FILE_NAME}: Enforce quota at N levels below abstract namespace" {
  # Put a ResourceQuota in the root abstract namespace demanding there can be at most 2 configmaps
  debug::log "Adding ResourceQuota to root abstract namespace with 'configmaps: 2'"
  git::add "${YAML_DIR}"/resourcequotas/resourcequota-configs.yaml acme/namespaces/rq-configmaps.yaml
  git::commit

  # Verify that this new resourcequota is set for all 5 child namespaces
  debug::log "Verifying ResourceQuota installed in all 5 child namespaces"
  for child_dir in "${CHILD_DIRS[@]}"; do
    wait::for -s -t 15 -o "2 0" -- \
      kubectl get resourcequota config-management-resource-quota -n="${child_dir}"  \
        -o=jsonpath="{.status['.hard.configmaps', '.used.configmaps']}"
  done

  debug::log "Creating 2 dummy configmaps, each in a different child namespace and different parent namespaces one level up"
  for child_dir in "${CHILD_DIRS[0]}" "${CHILD_DIRS[3]}"; do
    wait::for -s -t 15 -- \
      kubectl create configmap "${child_dir}" --from-literal=cat.fur=fluffy -n="${child_dir}" \
       && kubectl label configmap "${child_dir}" "${TEST_LABEL}" -n="${child_dir}"
  done

  debug::log "Attempted to add 3rd configmap, which should exceed the quota of 2"
  wait::for -f -t 15 -- \
    kubectl create configmap "${CHILD_DIRS[2]}" --from-literal=cat.fur=fluffy -n="${CHILD_DIRS[2]}"
}

@test "${FILE_NAME}: Enforce quota with multiple resources one level below abstract namespace" {
  debug::log "Adding ResourceQuota to root abstract namespace with 'configmaps: 1' and 'pods: 1'"
  git::add "${YAML_DIR}"/resourcequotas/resourcequota-configs-pods.yaml acme/namespaces/eng/rq-configmaps-pods.yaml
  git::commit

  debug::log "Verifying ResourceQuota installed in child namespaces of 'eng'"
  for child_dir in "${CHILD_DIRS[@]:0:3}"; do
    wait::for -s -t 15 -o "1 0 1 0" -- \
      kubectl get resourcequota config-management-resource-quota -n="${child_dir}"  \
        -o=jsonpath="{.status['.hard.configmaps', '.used.configmaps', '.hard.pods', '.used.pods']}"
  done

  # Create one pod and one config map in different namespaces
  debug::log "Creating a configmap in 1st child namespace"
    wait::for -s -t 15 -- \
      kubectl create configmap "${CHILD_DIRS[0]}" --from-literal=cat.fur=fluffy -n="${CHILD_DIRS[0]}" \
       && kubectl label configmap "${CHILD_DIRS[0]}" "${TEST_LABEL}" -n="${CHILD_DIRS[0]}"

  debug::log "Creating a pod in the 2nd child namespace"
  wait::for -s -t 15 -- \
    kubectl create -s "${YAML_DIR}"/pod-helloworld.yaml -n="${CHILD_DIRS[1]}" \
       && kubectl label pod hello-world "${TEST_LABEL}" -n="${CHILD_DIRS[1]}"

  debug::log "Attempting to create a configmap in the 3rd child namespace. This should exceed quota of 1"
  wait::for -f -t 15 -- \
    kubectl create configmap "${CHILD_DIRS[2]}" --from-literal=cat.fur=fluffy -n="${CHILD_DIRS[2]}"

  debug::log "Attempting to create a pod in 3rd child namespace. This should exceed quota of 1"
  wait::for -f -t 15 -- \
    kubectl create -f "${YAML_DIR}"/pod-helloworld.yaml -n="${CHILD_DIRS[2]}"
}

@test "${FILE_NAME}: Enforce quota for compute and memory resources" {
  debug::log "Adding ResourceQuota to abstract namespace 'eng' with 'limits.cpu: 2' and 'limits.memory: 100m'"
  git::add "${YAML_DIR}"/resourcequotas/resourcequota-compute.yaml acme/namespaces/rq-compute.yaml
  git::commit

  debug::log "Verifying ResourceQuota installed in child namespaces of 'eng'"
  for child_dir in "${CHILD_DIRS[@]:0:3}"; do
    wait::for -s -t 15 -o "2 0 100m 0" -- \
      kubectl get resourcequota config-management-resource-quota -n="${child_dir}"  \
        -o=jsonpath="{.status['.hard.limits\.cpu', '.used.limits\.cpu', '.hard.limits\.memory', '.used.limits\.memory']}"
  done

  debug::log "Creating a cpu hungry pod that is still within quota limits"
  wait::for -s -t 15 -- \
    kubectl create -s "${YAML_DIR}"/resourcequotas/pod-helloworld-cpu-hungry.yaml -n="${CHILD_DIRS[0]}" \
       && kubectl label pod hello-world-pod "${TEST_LABEL}" -n="${CHILD_DIRS[0]}"

  debug::log "Creating a memory hungry pod that is still within quota limits"
  wait::for -s -t 15 -- \
    kubectl create -s "${YAML_DIR}"/resourcequotas/pod-helloworld-mem-hungry.yaml -n="${CHILD_DIRS[1]}" \
       && kubectl label pod hello-world-pod "${TEST_LABEL}" -n="${CHILD_DIRS[1]}"

  debug::log "Creating another cpu hungry pod that should exceed cpu quota limits"
  wait::for -f -t 15 -- \
    kubectl create -s "${YAML_DIR}"/resourcequotas/pod-helloworld-cpu-hungry.yaml -n="${CHILD_DIRS[3]}"

  debug::log "Creating another memory hungry pod that should exceed memory quota limits"
  wait::for -f -t 15 -- \
    kubectl create -s "${YAML_DIR}"/resourcequotas/pod-helloworld-memory-hungry.yaml -n="${CHILD_DIRS[3]}"
}


