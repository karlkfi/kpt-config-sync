#!/bin/bash

set -euo pipefail

load ../lib/loader

# Project should exist
@test "Namespace create get delete" {
  test_create_get_delete "${GCP_TEST_PROJECT}"
}

@test "Namespace subfolder create get delete" {
  test_create_get_delete "${GCP_TEST_PROJECT_SUBFOLDER}"
}

function test_create_get_delete() {
  local ns="$1"
  echo "test_create_get_delete(${ns})"
  echo "The namespace should not exist initially"
  namespace::check_not_found "${ns}"

  echo "Create the namespace under the test project"
  run gcloud --quiet alpha container policy \
      namespaces create "${ns}" --project="${ns}"
  assert::contains "namespaces/${ns}"
  [ "$status" -eq 0 ]

  echo "See if namespace can be retrieved using describe"
  run gcloud --quiet alpha container policy \
      namespaces describe "${project}" --project="${ns}"
  assert::contains "namespaces/${ns}"
  [ "$status" -eq 0 ]

  echo "See if namespace can be listed by project"
  run gcloud --quiet alpha container policy \
      namespaces list --project="${ns}"
  assert::contains "namespaces/${ns}"
  [ "$status" -eq 0 ]

  echo "Wait to see if the namespace appears on the cluster"
  namespace::check_exists "${ns}"

  echo "Now delete the namespace"
  run gcloud --quiet alpha container policy \
      namespaces delete "${ns}" --project="${ns}"
  [ "$status" -eq 0 ]

  echo "See if namespace can no longer be described"
  run gcloud --quiet alpha container policy \
      namespaces describe "${ns}" --project="${ns}"
  assert::contains "NOT_FOUND"
  [ "$status" -eq 1 ]

  echo "The namespace should not exist on the cluster at the end"
  namespace::check_not_found "${ns}"
}

