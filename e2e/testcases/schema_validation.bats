#!/bin/bash
# Tests that schema validation prohibits invalid Nomos resource from being written.

set -euo pipefail

load "../lib/cluster"
load "../lib/debug"
load "../lib/setup"

YAML_DIR=${BATS_TEST_DIRNAME}/../testdata

function setup() {
  setup::common
}

# This cleans up any nomos resources that were erroneously written.
function teardown() {
  kubectl delete syncs invalid || true
  kubectl delete policynodes invalid || true
  kubectl delete clusterpolicies invalid || true
  setup::common_teardown
}

function test_invalid_yamls_for_resource() {
  local resource=${1}

  if cluster::check_minor_version_is_lt 11; then
    # Testing on a cluster that doesn't have schema validation
    skip
  fi

  for invalid_yaml in "${YAML_DIR}"/invalid/"${resource}"*.yaml; do
    debug::log "trying to update with ${invalid_yaml}"
    run kubectl apply -f "${invalid_yaml}"
    # shellcheck disable=SC2154
    if [ "$status" -eq 0 ]; then
      debug::error "successfully applied invalid manifest: ${invalid_yaml}"
    fi
  done
}

# TODO(sbochins): Make these three separate tests after we speed up setup and
# teardown for tests that do not depend on acme repo.
@test "Validation prevents invalid Nomos Resource writes" {
  test_invalid_yamls_for_resource "policynode"
  test_invalid_yamls_for_resource "clusterpolicy"
  test_invalid_yamls_for_resource "sync"
}
