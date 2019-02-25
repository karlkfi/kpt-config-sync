#!/bin/bash
#
# This testcase exists to verify that the "acme" org is created fully.
#

set -euo pipefail

load "../lib/assert"
load "../lib/debug"
load "../lib/namespace"
load "../lib/resource"
load "../lib/setup"
load "../lib/wait"

declare KUBE_PROXY_PID
setup() {
  setup::common
  setup::git::initialize
  setup::git::init_acme
  kubectl proxy &
  KUBE_PROXY_PID=$!
}

function teardown() {
  kill $KUBE_PROXY_PID
  wait $KUBE_PROXY_PID || true
  setup::common_teardown
}

function check_metrics_pages() {
  local service="${1:-}"

  local base_url="localhost:8001/api/v1/namespaces/config-management-system/services"
  local port="metrics/proxy"
  local output
  output="$(curl ${base_url}/${service}:${port}/threads 2>&1)" \
    || debug::error "Failed to scrape ${service} /threads: ${output}"
  output="$(curl ${base_url}/${service}:${port}/metrics 2>&1)" \
    || debug::error "Failed to scrape ${service} /metrics: ${output}"
}

@test "All acme corp created" {
  local ns

  resource::check_count -r namespace -a "config.gke.io/managed=enabled" -c ${#ACME_NAMESPACES[@]}

  # Cluster-scoped resources
  namespace::check_exists analytics \
    -l "config.gke.io/quota=true" \
    -a "config.gke.io/managed=enabled"
  namespace::check_exists backend \
    -l "config.gke.io/quota=true" \
    -a "config.gke.io/managed=enabled"
  namespace::check_exists frontend \
    -l "config.gke.io/quota=true" \
    -a "config.gke.io/managed=enabled"
  namespace::check_exists new-prj \
    -l "config.gke.io/quota=true" \
    -a "config.gke.io/managed=enabled"
  namespace::check_exists newer-prj \
    -l "config.gke.io/quota=true" \
    -a "config.gke.io/managed=enabled"
  resource::check_count -r validatingwebhookconfigurations -c 1
  resource::check validatingwebhookconfigurations resource-quota.nomos.dev
  resource::check_count -a "config.gke.io/managed=enabled" -r clusterrole -c 3
  resource::check clusterrole acme-admin -a "config.gke.io/managed=enabled"
  resource::check clusterrole namespace-viewer -a "config.gke.io/managed=enabled"
  resource::check clusterrole rbac-viewer -a "config.gke.io/managed=enabled"
  resource::check_count -a "config.gke.io/managed=enabled" -r clusterrolebinding -c 2
  resource::check clusterrolebinding namespace-viewers -a "config.gke.io/managed=enabled"
  resource::check clusterrolebinding rbac-viewers -a "config.gke.io/managed=enabled"
  resource::check_count -r podsecuritypolicy -c 1
  resource::check podsecuritypolicy example -a "config.gke.io/managed=enabled"

  # Namespace-scoped resources
  # analytics
  resource::check_count -n analytics -r role -c 0
  resource::check_count -n analytics -r rolebinding -c 2
  resource::check -n analytics rolebinding mike-rolebinding -a "config.gke.io/managed=enabled"
  resource::check -n analytics rolebinding alice-rolebinding -a "config.gke.io/managed=enabled"
  resource::check_count -n analytics -r resourcequota -c 1 -a "config.gke.io/managed=enabled"
  resource::check -n analytics resourcequota nomos-resource-quota -a "config.gke.io/managed=enabled"

  # backend
  resource::check_count -n backend -r role -c 0
  resource::check_count -n backend -r rolebinding -c 2
  resource::check -n backend rolebinding bob-rolebinding -a "config.gke.io/managed=enabled"
  resource::check -n backend rolebinding alice-rolebinding -a "config.gke.io/managed=enabled"
  resource::check_count -n backend -r resourcequota -c 1 -a "config.gke.io/managed=enabled"
  resource::check -n backend resourcequota nomos-resource-quota -a "config.gke.io/managed=enabled"
  run kubectl get quota -n backend -o yaml
  assert::contains 'pods: "1"'

  # frontend
  namespace::check_exists frontend -l "env=prod" -a "audit=true"
  resource::check_count -n frontend -r role -c 0
  resource::check_count -n frontend -r rolebinding -c 2
  resource::check -n frontend rolebinding alice-rolebinding -a "config.gke.io/managed=enabled"
  resource::check -n frontend rolebinding sre-admin -a "config.gke.io/managed=enabled"
  resource::check_count -n frontend -r resourcequota -c 1 -a "config.gke.io/managed=enabled"
  resource::check -n frontend resourcequota nomos-resource-quota -a "config.gke.io/managed=enabled"

  # new-prj
  resource::check_count -n new-prj -r role -c 1
  resource::check -n new-prj role acme-admin -a "config.gke.io/managed=enabled"
  resource::check_count -n new-prj -r rolebinding -c 0
  resource::check_count -n new-prj -r resourcequota -c 1 -a "config.gke.io/managed=enabled"
  resource::check -n new-prj resourcequota nomos-resource-quota -a "config.gke.io/managed=enabled"

  # newer-prj
  resource::check_count -n newer-prj -r role -c 0
  resource::check_count -n newer-prj -r rolebinding -c 0
  resource::check_count -n newer-prj -r resourcequota -c 1 -a "config.gke.io/managed=enabled"
  resource::check -n newer-prj resourcequota nomos-resource-quota -a "config.gke.io/managed=enabled"

  local services=(
    git-policy-importer
    resourcequota-admission-controller
    syncer
  )
  for service in "${services[@]}"; do
    check_metrics_pages ${service}
  done

}

