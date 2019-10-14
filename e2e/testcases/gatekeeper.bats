#!/bin/bash

set -euo pipefail

load "../lib/debug"
load "../lib/git"
load "../lib/nomos"
load "../lib/setup"
load "../lib/wait"

YAML_DIR=${BATS_TEST_DIRNAME}/../testdata/gatekeeper

setup() {
  setup::common
  setup::git::initialize

  local TEST_REPO_DIR=${BATS_TMPDIR}
  cd "${TEST_REPO_DIR}/repo"

  mkdir -p acme/system
  cp -r /opt/testing/e2e/examples/acme/system acme
  git add -A
  git::commit -a -m "Commit minimal repo contents."

  wait::for -t 30 -- nomos::repo_synced
}

function teardown() {
  kubectl delete -f "${YAML_DIR}/constraint-template-crd.yaml" --ignore-not-found
  kubectl delete -f "${YAML_DIR}/constraint-crd.yaml" --ignore-not-found
  setup::git::remove_all acme
  setup::common_teardown
}

@test "constraint template and constraint in same commit" {
  debug::log "Adding gatekeeper CT CRD"
  kubectl apply -f "${YAML_DIR}/constraint-template-crd.yaml"

  debug::log "Adding CT/C in one commit"
  git::add \
    "${YAML_DIR}/constraint-template.yaml" \
    acme/cluster/constraint-template.yaml
  git::add \
    "${YAML_DIR}/constraint.yaml" \
    acme/cluster/constraint.yaml
  git::commit

  debug::log "Waiting for CT on API server"
  wait::for -t 30 -- kubectl get constrainttemplate k8sallowedrepos

  debug::log "Applying CRD for constraint (simulates gatekeeper)"
  kubectl apply -f "${YAML_DIR}/constraint-crd.yaml"

  debug::log "Waiting for nomos to recover"
  wait::for -t 60 -- nomos::repo_synced
}
