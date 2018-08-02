#!/bin/bash

set -euo pipefail

load ../lib/loader

# Project should exist
@test "Namespace under org create/get/delete" {
  test_create_get_delete "${GCP_TEST_PROJECT}"
}

@test "Namespace under folder create/get/delete" {
  test_create_get_delete "${GCP_TEST_PROJECT_SUBFOLDER}"
}

@test "Project IAM create/delete" {
  echo "The namespace should not exist initially"
  namespace::check_not_found "${GCP_TEST_PROJECT}"

  echo "Create the namespace under the test project"
  run gcloud --quiet alpha container policy \
      namespaces create "${GCP_TEST_PROJECT}" \
      --project="${GCP_TEST_PROJECT}"
  assert::contains "namespaces/${GCP_TEST_PROJECT}"
  [ "$status" -eq 0 ]

  echo "Wait to see if the namespace appears on the cluster"
  namespace::check_exists "${GCP_TEST_PROJECT}"

  echo "Adding IAM binding to project"
  run gcloud projects add-iam-policy-binding "${GCP_TEST_PROJECT}" \
      --member=user:bob@nomos-e2e.joonix.net --role=roles/container.viewer
  [ "$status" -eq 0 ]

  echo "Checking for binding to be created in namespace"
  wait::for -- kubectl get configmaps -n "${GCP_TEST_PROJECT}" \
      --as bob@nomos-e2e.joonix.net
  resource::check -n "${GCP_TEST_PROJECT}" rolebinding \
      "${GCP_TEST_PROJECT}.${GCP_TEST_PROJECT}.container.viewer"

  echo "Removing IAM binding from project"
  run gcloud projects remove-iam-policy-binding "${GCP_TEST_PROJECT}" \
      --member=user:bob@nomos-e2e.joonix.net --role=roles/container.viewer

  echo "Checking for binding to be removed from namespace"
  wait::for -f -- kubectl get configmaps -n "${GCP_TEST_PROJECT}" \
      --as bob@nomos-e2e.joonix.net
}

@test "Project move to folder with IAM" {
  echo "The namespace should not exist initially"
  namespace::check_not_found "${GCP_TEST_PROJECT}"

  echo "Create the namespace under the test project"
  run gcloud --quiet alpha container policy \
      namespaces create "${GCP_TEST_PROJECT}" \
      --project="${GCP_TEST_PROJECT}"
  assert::contains "namespaces/${GCP_TEST_PROJECT}"
  [ "$status" -eq 0 ]

  echo "Wait to see if the namespace appears on the cluster"
  namespace::check_exists "${GCP_TEST_PROJECT}"

  echo "Adding IAM binding to folder"
  run gcloud alpha resource-manager folders add-iam-policy-binding "${FOLDER_ID}" \
      --member=user:bob@nomos-e2e.joonix.net --role=roles/container.viewer
  [ "$status" -eq 0 ]

  echo "Moving project into folder"
  run gcloud alpha projects move "${GCP_TEST_PROJECT}" --folder "${FOLDER_ID}"

  echo "Checking for binding to be created in namespace"
  wait::for -- kubectl get configmaps -n "${GCP_TEST_PROJECT}" \
      --as bob@nomos-e2e.joonix.net
  resource::check -n "${GCP_TEST_PROJECT}" rolebinding \
      "${GCP_TEST_PROJECT}.folders-${FOLDER_ID}.container.viewer"

  echo "Moving project back to org"
  run gcloud alpha projects move "${GCP_TEST_PROJECT}" \
      --organization "${GCP_TEST_ORGANIZATION_ID}"

  echo "Checking for binding to be removed from namespace"
  wait::for -f -- kubectl get configmaps -n "${GCP_TEST_PROJECT}" \
      --as bob@nomos-e2e.joonix.net
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
      namespaces describe "${ns}" --project="${ns}"
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

