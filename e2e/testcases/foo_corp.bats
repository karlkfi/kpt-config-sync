#!/bin/bash
#
# This testcase exists to verify that the "foo-corp" org is created fully.
#

set -euo pipefail

load "../lib/git"
load "../lib/namespace"
load "../lib/nomos"
load "../lib/resource"
load "../lib/setup"

FILE_NAME="$(basename "${BATS_TEST_FILENAME}" '.bats')"

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

@test "${FILE_NAME}: All foo-corp created" {
  # TODO(frankf): POLICY_DIR is currently set to "acme" during installation.
  # This should be resolved with new repo format.
  git::add /opt/testing/e2e/examples/foo-corp-example/foo-corp acme
  git::commit

  wait::for -t 180 -- nomos::repo_synced

  local ns
  for ns in "${FOOCORP_NAMESPACES[@]}"; do
    namespace::check_exists "$ns" \
      -l "configmanagement.gke.io/quota=true" \
      -a "configmanagement.gke.io/managed=enabled"
  done

  resource::check_count -r namespace -l "configmanagement.gke.io/quota=true" -a "configmanagement.gke.io/managed=enabled" -c ${#FOOCORP_NAMESPACES[@]}
  resource::check_count -a "configmanagement.gke.io/managed=enabled" -r clusterrole -c 2
  resource::check clusterrole namespace-reader -a "configmanagement.gke.io/managed=enabled"
  resource::check clusterrole pod-creator -a "configmanagement.gke.io/managed=enabled"
  resource::check_count -a "configmanagement.gke.io/managed=enabled" -r clusterrolebinding -c 1
  resource::check clusterrolebinding namespace-readers -a "configmanagement.gke.io/managed=enabled"
  resource::check customresourcedefinition fulfillmentcenters.foo-corp.com -a "configmanagement.gke.io/managed=enabled"

  # Namespace-scoped resources
  # audit
  resource::check_count -n audit -r role -c 0
  resource::check_count -n audit -r rolebinding -c 1
  resource::check -n audit rolebinding viewers -a "configmanagement.gke.io/managed=enabled"
  resource::check_count -n audit -r resourcequota -c 0 -a "configmanagement.gke.io/managed=enabled"

  # shipping-dev
  resource::check_count -n shipping-dev -r role -c 1
  resource::check -n shipping-dev role job-creator -a "configmanagement.gke.io/managed=enabled"
  resource::check_count -n shipping-dev -r rolebinding -c 3
  resource::check -n shipping-dev rolebinding viewers -a "configmanagement.gke.io/managed=enabled"
  resource::check -n shipping-dev rolebinding pod-creators -a "configmanagement.gke.io/managed=enabled"
  resource::check -n shipping-dev rolebinding job-creators -a "configmanagement.gke.io/managed=enabled"
  resource::check_count -n shipping-dev -r resourcequota -c 1 -a "configmanagement.gke.io/managed=enabled"
  resource::check -n shipping-dev resourcequota quota -a "configmanagement.gke.io/managed=enabled"

  # shipping-staging
  resource::check_count -n shipping-staging -r role -c 0
  resource::check_count -n shipping-staging -r rolebinding -c 2
  resource::check -n shipping-staging rolebinding viewers -a "configmanagement.gke.io/managed=enabled"
  resource::check -n shipping-staging rolebinding pod-creators -a "configmanagement.gke.io/managed=enabled"
  resource::check_count -n shipping-staging -r resourcequota -c 1 -a "configmanagement.gke.io/managed=enabled"
  resource::check -n shipping-staging resourcequota quota -a "configmanagement.gke.io/managed=enabled"
  resource::check_count -n shipping-staging -r fulfillmentcenter -c 1 -a "configmanagement.gke.io/managed=enabled"

  # shipping-prod
  resource::check_count -n shipping-prod -r role -c 0
  resource::check_count -n shipping-prod -r rolebinding -c 3
  resource::check -n shipping-prod rolebinding viewers -a "configmanagement.gke.io/managed=enabled"
  resource::check -n shipping-prod rolebinding pod-creators -a "configmanagement.gke.io/managed=enabled"
  resource::check -n shipping-prod rolebinding sre-admin -a "configmanagement.gke.io/managed=enabled"
  resource::check_count -n shipping-prod -r resourcequota -c 1 -a "configmanagement.gke.io/managed=enabled"
  resource::check -n shipping-prod resourcequota quota -a "configmanagement.gke.io/managed=enabled"
  resource::check_count -n shipping-prod -r fulfillmentcenter -c 1 -a "configmanagement.gke.io/managed=enabled"

  setup::git::remove_all acme
}

