#!/bin/bash
#
# This testcase exists to verify that the "foo-corp" org is created fully.
#

set -euo pipefail

load "../lib/git"
load "../lib/namespace"
load "../lib/resource"
load "../lib/setup"

setup() {
  setup::common
  setup::git::initialize
}

teardown() {
  setup::common_teardown
}

FOOCORP_NAMESPACES=(
  audit
  shipping-dev
  shipping-prod
  shipping-staging
)

@test "All foo-corp created" {
  # TODO(frankf): POLICY_DIR is currently set to "acme" during installation.
  # This should be resolved with new repo format.
  git::add /opt/testing/e2e/examples/foo-corp-example/foo-corp acme
  git::commit

  local ns
  local commit_hash
  commit_hash="$(git::hash)"
  for ns in "${FOOCORP_NAMESPACES[@]}"; do
    wait::for -- policynode::sync_token_eq "${ns}" "${commit_hash}"
  done
  for ns in "${ACME_NAMESPACES[@]}"; do
    wait::for -f -- kubectl get ns "${ns}"
  done

  # Cluster-scoped resources
  namespace::check_exists audit \
    -l "nomos.dev/managed=enabled" \
    -a "nomos.dev/managed=enabled"
  namespace::check_exists shipping-dev \
    -l "nomos.dev/managed=enabled" \
    -a "nomos.dev/managed=enabled"
  namespace::check_exists shipping-staging \
    -l "nomos.dev/managed=enabled" \
    -a "nomos.dev/managed=enabled"
  namespace::check_exists shipping-prod \
    -l "nomos.dev/managed=enabled" \
    -a "nomos.dev/managed=enabled"
  resource::check_count -r namespace -l "nomos.dev/managed=enabled" -a "nomos.dev/managed=enabled" -c 4
  resource::check_count -a "nomos.dev/managed=enabled" -r clusterrole -c 2
  resource::check clusterrole namespace-reader -a "nomos.dev/managed=enabled"
  resource::check clusterrole pod-creator -a "nomos.dev/managed=enabled"
  resource::check_count -a "nomos.dev/managed=enabled" -r clusterrolebinding -c 1
  resource::check clusterrolebinding namespace-readers -a "nomos.dev/managed=enabled"

  # Namespace-scoped resources
  # audit
  resource::check_count -n audit -r role -c 0
  resource::check_count -n audit -r rolebinding -c 1
  resource::check -n audit rolebinding viewers -a "nomos.dev/managed=enabled"
  resource::check_count -n audit -r resourcequota -c 0 -a "nomos.dev/managed=enabled"

  # shipping-dev
  resource::check_count -n shipping-dev -r role -c 1
  resource::check -n shipping-dev role job-creator -a "nomos.dev/managed=enabled"
  resource::check_count -n shipping-dev -r rolebinding -c 3
  resource::check -n shipping-dev rolebinding viewers -a "nomos.dev/managed=enabled"
  resource::check -n shipping-dev rolebinding pod-creators -a "nomos.dev/managed=enabled"
  resource::check -n shipping-dev rolebinding job-creators -a "nomos.dev/managed=enabled"
  resource::check_count -n shipping-dev -r resourcequota -c 1 -a "nomos.dev/managed=enabled"
  resource::check -n shipping-dev resourcequota nomos-resource-quota -a "nomos.dev/managed=enabled"

  # shipping-staging
  resource::check_count -n shipping-staging -r role -c 0
  resource::check_count -n shipping-staging -r rolebinding -c 2
  resource::check -n shipping-staging rolebinding viewers -a "nomos.dev/managed=enabled"
  resource::check -n shipping-staging rolebinding pod-creators -a "nomos.dev/managed=enabled"
  resource::check_count -n shipping-staging -r resourcequota -c 1 -a "nomos.dev/managed=enabled"
  resource::check -n shipping-staging resourcequota nomos-resource-quota -a "nomos.dev/managed=enabled"

  # shipping-prod
  resource::check_count -n shipping-prod -r role -c 0
  resource::check_count -n shipping-prod -r rolebinding -c 3
  resource::check -n shipping-prod rolebinding viewers -a "nomos.dev/managed=enabled"
  resource::check -n shipping-prod rolebinding pod-creators -a "nomos.dev/managed=enabled"
  resource::check -n shipping-prod rolebinding sre-admin -a "nomos.dev/managed=enabled"
  resource::check_count -n shipping-prod -r resourcequota -c 1 -a "nomos.dev/managed=enabled"
  resource::check -n shipping-prod resourcequota nomos-resource-quota -a "nomos.dev/managed=enabled"
}

