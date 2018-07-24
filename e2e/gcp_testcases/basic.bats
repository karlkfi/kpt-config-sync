#!/bin/bash

set -euo pipefail

load ../lib/loader

@test "Namespace create, test project must already exist" {
  echo "The namespace should not exist initially"
  namespace::check_not_found "${GCP_TEST_NAMESPACE}"

  echo "Create the namespace under the test project"
  wait::for -t 15 -- gcloud --quiet alpha container policy \
      namespaces create "${GCP_TEST_NAMESPACE}" --project="${GCP_TEST_PROJECT}"

  echo "Wait to see if the namespace appears on the cluster"
  namespace::check_exists "${GCP_TEST_NAMESPACE}" -t 20

  echo "Now delete the namespace"
  wait::for -t 15 -- gcloud --quiet alpha container policy \
      namespaces delete "${GCP_TEST_NAMESPACE}" --project="${GCP_TEST_PROJECT}"

  echo "The namespace should not exist at the end"
  namespace::check_not_found "${GCP_TEST_NAMESPACE}"
}
