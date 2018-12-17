#!/bin/bash
#
# This testcase exists to verify that the "foo-corp" org is created fully.
#

set -euo pipefail

load "../lib/assert"
load "../lib/cluster"
load "../lib/debug"
load "../lib/git"
load "../lib/ignore"
load "../lib/namespace"
load "../lib/policynode"
load "../lib/resource"
load "../lib/setup"
load "../lib/wait"

@test "All foo-corp created" {
  git::rm acme
  # TODO(frankf): POLICY_DIR is currently set to "acme" during installation.
  # This should be resolved with new repo format.
  git::add /opt/testing/e2e/examples/foo-corp-example/foo-corp acme
  git::commit

  # TODO(117440556): Wait for reconciliation to be complete.
  sleep 15

  # Cluster-scoped resources
  namespace::check_exists audit \
    -l "nomos.dev/managed=enabled"
  namespace::check_exists shipping-dev \
    -l "nomos.dev/managed=enabled"
  namespace::check_exists shipping-staging \
    -l "nomos.dev/managed=enabled"
  namespace::check_exists shipping-prod \
    -l "nomos.dev/managed=enabled"
  resource::check_count -r namespace -l nomos.dev/managed -c 4
  resource::check_count -l "nomos.dev/managed=enabled" -r clusterrole -c 2
  resource::check clusterrole namespace-reader -l "nomos.dev/managed=enabled"
  resource::check clusterrole pod-creator -l "nomos.dev/managed=enabled"
  resource::check_count -l "nomos.dev/managed=enabled" -r clusterrolebinding -c 1
  resource::check clusterrolebinding namespace-readers -l "nomos.dev/managed=enabled"

  # Namespace-scoped resources
  # audit
  resource::check_count -n audit -r role -c 0
  resource::check_count -n audit -r rolebinding -c 1
  resource::check -n audit rolebinding viewers -l "nomos.dev/managed=enabled"
  resource::check_count -n audit -r resourcequota -c 0

  # shipping-dev
  resource::check_count -n shipping-dev -r role -c 1
  resource::check -n shipping-dev role job-creator -l "nomos.dev/managed=enabled"
  resource::check_count -n shipping-dev -r rolebinding -c 3
  resource::check -n shipping-dev rolebinding viewers -l "nomos.dev/managed=enabled"
  resource::check -n shipping-dev rolebinding pod-creators -l "nomos.dev/managed=enabled"
  resource::check -n shipping-dev rolebinding job-creators -l "nomos.dev/managed=enabled"
  resource::check_count -n shipping-dev -r resourcequota -c 1
  resource::check -n shipping-dev resourcequota nomos-resource-quota -l "nomos.dev/managed=enabled"

  # shipping-staging
  resource::check_count -n shipping-staging -r role -c 0
  resource::check_count -n shipping-staging -r rolebinding -c 2
  resource::check -n shipping-staging rolebinding viewers -l "nomos.dev/managed=enabled"
  resource::check -n shipping-staging rolebinding pod-creators -l "nomos.dev/managed=enabled"
  resource::check_count -n shipping-staging -r resourcequota -c 1
  resource::check -n shipping-staging resourcequota nomos-resource-quota -l "nomos.dev/managed=enabled"

  # shipping-prod
  resource::check_count -n shipping-prod -r role -c 0
  resource::check_count -n shipping-prod -r rolebinding -c 3
  resource::check -n shipping-prod rolebinding viewers -l "nomos.dev/managed=enabled"
  resource::check -n shipping-prod rolebinding pod-creators -l "nomos.dev/managed=enabled"
  resource::check -n shipping-prod rolebinding sre-admin -l "nomos.dev/managed=enabled"
  resource::check_count -n shipping-prod -r resourcequota -c 1
  resource::check -n shipping-prod resourcequota nomos-resource-quota -l "nomos.dev/managed=enabled"
}

