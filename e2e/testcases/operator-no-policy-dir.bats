#!/bin/bash

set -euo pipefail

load "../lib/git"
load "../lib/configmap"
load "../lib/setup"
load "../lib/wait"

FILE_NAME="$(basename "${BATS_TEST_FILENAME}" '.bats')"

setup() {
  setup::common
  setup::git::initialize
  setup::git::init_acme
}

teardown() {
  kubectl apply -f "${BATS_TEST_DIRNAME}/../operator-config-git.yaml"
  setup::common_teardown
}


@test "${FILE_NAME}: Absence of cluster, namespaces, systems directories at POLICY_DIR yields a missing repo error" {
  kubectl apply -f "${BATS_TEST_DIRNAME}/../operator-config-git-no-policy-dir.yaml"

  # Verify that the application of the operator config yields the correct error code
  wait::for -t 60 -o "KNV1017" -- kubectl get repo repo -o=jsonpath='{.status.import.errors[0].code}'
}

# This test case gradually adjusts root and acme directories' contents to move from:
# POLICY_DIR == "acme" to: POLICY_DIR undefined in the template.  Thus the gradual steps.
@test "${FILE_NAME}: Confirm that config-management-operator starts correctly with POLICY_DIR unset" {
  setup::git::add_acme_contents_to_root

  kubectl apply -f "${BATS_TEST_DIRNAME}/../operator-config-git-no-policy-dir.yaml"

  setup::git::remove_acme_folder

  wait::for -t 30 -- test "Running" == "$( \
    kubectl get pods \
      -n kube-system \
      -o json \
      -l "component=config-management-operator" \
      -o=jsonpath='{.items[0].status.phase}' \
  )"

  wait::for -t 60 -o "" -- kubectl get repo repo -o=jsonpath='{.status.import.errors}'
}
