#!/bin/bash

set -euo pipefail

load ../lib/loader

# Project should exist
@test "Namespace create get delete" {
  echo "The namespace should not exist initially"
  namespace::check_not_found "${GCP_TEST_NAMESPACE}"

  echo "Create the namespace under the test project"
  run gcloud --quiet alpha container policy \
      namespaces create "${GCP_TEST_NAMESPACE}" --project="${GCP_TEST_PROJECT}"
  assert::contains "namespaces/${GCP_TEST_NAMESPACE}"
  [ "$status" -eq 0 ]

  echo "See if namespace can be retrieved using describe"
  run gcloud --quiet alpha container policy \
      namespaces describe "${GCP_TEST_NAMESPACE}" --project="${GCP_TEST_PROJECT}"
  assert::contains "namespaces/${GCP_TEST_NAMESPACE}"
  [ "$status" -eq 0 ]

  echo "See if namespace can be listed by project"
  run gcloud --quiet alpha container policy \
      namespaces list --project="${GCP_TEST_PROJECT}"
  assert::contains "namespaces/${GCP_TEST_NAMESPACE}"
  [ "$status" -eq 0 ]

  echo "Wait to see if the namespace appears on the cluster"
  namespace::check_exists "${GCP_TEST_NAMESPACE}"

  echo "Now delete the namespace"
  run gcloud --quiet alpha container policy \
      namespaces delete "${GCP_TEST_NAMESPACE}" --project="${GCP_TEST_PROJECT}"
  [ "$status" -eq 0 ]

  echo "See if namespace can no longer be described"
  run gcloud --quiet alpha container policy \
      namespaces describe "${GCP_TEST_NAMESPACE}" --project="${GCP_TEST_PROJECT}"
  assert::contains "NOT_FOUND"
  [ "$status" -eq 1 ]

  echo "The namespace should not exist on the cluster at the end"
  namespace::check_not_found "${GCP_TEST_NAMESPACE}"
}
