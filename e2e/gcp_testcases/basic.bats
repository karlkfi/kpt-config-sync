#!/bin/bash

set -euo pipefail

load ../lib/loader

# Project should exist
@test "Namespace under org create/get/delete" {
  test_create_get_delete "${GCP_PROJECT_B}"
}

@test "Namespace under folder create/get/delete" {
  test_create_get_delete "${GCP_PROJECT_A}"
}

@test "Project IAM binding create/delete" {
  echo "The namespace should not exist initially"
  namespace::check_not_found "${GCP_PROJECT_B}"

  echo "Create the namespace under the test project"
  run gcloud --quiet alpha container policy \
      namespaces create "${GCP_PROJECT_B}" \
      --project="${GCP_PROJECT_B}"
  assert::contains "namespaces/${GCP_PROJECT_B}"
  [ "$status" -eq 0 ]

  echo "Wait to see if the namespace appears on the cluster"
  namespace::check_exists "${GCP_PROJECT_B}"

  echo "Adding IAM binding to project"
  run gcloud projects add-iam-policy-binding "${GCP_PROJECT_B}" \
      --member=user:bob@nomos-e2e.joonix.net --role=roles/container.viewer
  [ "$status" -eq 0 ]

  echo "Checking for binding to be created in namespace"
  wait::for -- kubectl get rolebinding \
      "${GCP_PROJECT_B}.container.viewer" \
      -n "${GCP_PROJECT_B}"
  run kubectl get configmaps -n "${GCP_PROJECT_B}" \
      --as bob@nomos-e2e.joonix.net

  echo "Removing IAM binding from project"
  run gcloud projects remove-iam-policy-binding "${GCP_PROJECT_B}" \
      --member=user:bob@nomos-e2e.joonix.net --role=roles/container.viewer

  echo "Checking for binding to be removed from namespace"
  wait::for -f -- kubectl get configmaps -n "${GCP_PROJECT_B}" \
      --as bob@nomos-e2e.joonix.net
}

@test "Folder IAM binding create/delete" {
  echo "The namespace should not exist initially"
  namespace::check_not_found "${GCP_PROJECT_A}"

  echo "Create the namespace under the test project"
  run gcloud --quiet alpha container policy \
      namespaces create "${GCP_PROJECT_A}" \
      --project="${GCP_PROJECT_A}"
  assert::contains "namespaces/${GCP_PROJECT_A}"
  [ "$status" -eq 0 ]

  echo "Wait to see if the namespace appears on the cluster"
  namespace::check_exists "${GCP_PROJECT_A}"

  echo "Adding IAM binding to folder"
  run gcloud alpha resource-manager folders add-iam-policy-binding "${FOLDER_ID}" \
      --member=user:bob@nomos-e2e.joonix.net --role=roles/container.viewer
  [ "$status" -eq 0 ]

  echo "Checking for binding to be created in namespace"
  wait::for -- kubectl get rolebinding \
      "folders-${FOLDER_ID}.container.viewer" \
      -n "${GCP_PROJECT_A}"
  run kubectl get configmaps -n "${GCP_PROJECT_A}" \
      --as bob@nomos-e2e.joonix.net
  [ "$status" -eq 0 ]

  echo "Removing IAM binding from folder"
  run gcloud alpha resource-manager folders remove-iam-policy-binding "${FOLDER_ID}" \
      --member=user:bob@nomos-e2e.joonix.net --role=roles/container.viewer || true

  echo "Checking for binding to be removed from namespace"
  wait::for -f -- kubectl get configmaps -n "${GCP_PROJECT_A}" \
      --as bob@nomos-e2e.joonix.net
}

@test "Default IAM bindings" {
  echo "The namespace should not exist initially"
  namespace::check_not_found "${GCP_PROJECT_A}"

  echo "Create the namespace under the test project"
  run gcloud --quiet alpha container policy \
      namespaces create "${GCP_PROJECT_A}" \
      --project="${GCP_PROJECT_A}"
  assert::contains "namespaces/${GCP_PROJECT_A}"

  echo "Wait to see if the namespace appears on the cluster"
  namespace::check_exists "${GCP_PROJECT_A}"

  wait::for -- kubectl get rolebinding \
      "organizations-${GCP_ORG_ID}.owner" \
      -n "${GCP_PROJECT_A}"
  wait::for -- kubectl get rolebinding \
      "${GCP_PROJECT_A}.owner" \
      -n "${GCP_PROJECT_A}"
  resource::check_count -r rolebinding -c 2 -n "${GCP_PROJECT_A}"
}

@test "Project move to folder" {
  echo "The namespace should not exist initially"
  namespace::check_not_found "${GCP_PROJECT_B}"

  echo "Create the namespace under the test project"
  run gcloud --quiet alpha container policy \
      namespaces create "${GCP_PROJECT_B}" \
      --project="${GCP_PROJECT_B}"
  assert::contains "namespaces/${GCP_PROJECT_B}"
  [ "$status" -eq 0 ]

  echo "Wait to see if the namespace appears on the cluster"
  namespace::check_exists "${GCP_PROJECT_B}"

  echo "Adding IAM binding to folder"
  run gcloud alpha resource-manager folders add-iam-policy-binding "${FOLDER_ID}" \
      --member=user:bob@nomos-e2e.joonix.net --role=roles/container.viewer
  [ "$status" -eq 0 ]

  echo "Moving project into folder"
  run gcloud alpha projects move "${GCP_PROJECT_B}" --folder "${FOLDER_ID}"

  echo "Checking for binding to be created in namespace"
  wait::for -- kubectl get rolebinding \
      "folders-${FOLDER_ID}.container.viewer" \
      -n "${GCP_PROJECT_B}"
  run kubectl get configmaps -n "${GCP_PROJECT_B}" \
      --as bob@nomos-e2e.joonix.net
  [ "$status" -eq 0 ]

  echo "Moving project back to org"
  run gcloud alpha projects move "${GCP_PROJECT_B}" \
      --organization "${GCP_ORG_ID}"

  echo "Checking for binding to be removed from namespace"
  wait::for -f -- kubectl get configmaps -n "${GCP_PROJECT_B}" \
      --as bob@nomos-e2e.joonix.net
}

@test "Resource quota not managed" {
  echo "Create the namespace under the test project"
  run gcloud --quiet alpha container policy \
      namespaces create "${GCP_PROJECT_A}" \
      --project="${GCP_PROJECT_A}"
  assert::contains "namespaces/${GCP_PROJECT_A}"
  [ "$status" -eq 0 ]

  echo "Wait to see if the namespace appears on the cluster"
  namespace::check_exists "${GCP_PROJECT_A}"

  wait::for -- kubectl get -n "${GCP_PROJECT_A}" resourcequota pod-quota &
  local pid=$!

  echo "Creating resource quota"
  kubectl apply -f - << EOF
kind: ResourceQuota
apiVersion: v1
metadata:
  name: pod-quota
  namespace: ${GCP_PROJECT_A}
spec:
  hard:
    pods: "1"
    secrets: "5"
EOF

  echo "Waiting for get quota to complete"
  wait "$pid"
  sleep 1

  echo "Removing resource quota"
  kubectl delete -n "${GCP_PROJECT_A}" resourcequota pod-quota
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

