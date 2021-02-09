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

KUBE_PROXY_PID=""

FILE_NAME="$(basename "${BATS_TEST_FILENAME}" '.bats')"

# Total count of namespaces in acme
ACME_NAMESPACES=(
  analytics
  backend
  frontend
  new-prj
  newer-prj
)

test_setup() {
  setup::git::initialize
  setup::git::init acme
  kubectl proxy &
  KUBE_PROXY_PID=$!
}

test_teardown() {
  kill $KUBE_PROXY_PID
  wait $KUBE_PROXY_PID || true
  setup::git::remove_all acme
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

@test "${FILE_NAME}: All acme corp created" {
  resource::check_count -r namespace \
   -a "configmanagement.gke.io/managed=enabled" \
   -a "hnc.x-k8s.io/managedBy=configmanagement.gke.io" \
   -a "hnc.x-k8s.io/managed-by=configmanagement.gke.io" \
   -c ${#ACME_NAMESPACES[@]}

  # Cluster-scoped resources
  namespace::check_exists analytics \
    -a "configmanagement.gke.io/managed=enabled" \
    -a "hnc.x-k8s.io/managedBy=configmanagement.gke.io" \
    -a "hnc.x-k8s.io/managed-by=configmanagement.gke.io" \
    -l "analytics.tree.hnc.x-k8s.io/depth=0" \
    -l "eng.tree.hnc.x-k8s.io/depth=1"
  namespace::check_exists backend \
    -a "configmanagement.gke.io/managed=enabled" \
    -a "hnc.x-k8s.io/managedBy=configmanagement.gke.io" \
    -a "hnc.x-k8s.io/managed-by=configmanagement.gke.io" \
    -l "backend.tree.hnc.x-k8s.io/depth=0" \
    -l "eng.tree.hnc.x-k8s.io/depth=1"
  namespace::check_exists frontend \
    -a "configmanagement.gke.io/managed=enabled" \
    -a "hnc.x-k8s.io/managedBy=configmanagement.gke.io" \
    -a "hnc.x-k8s.io/managed-by=configmanagement.gke.io" \
    -l "frontend.tree.hnc.x-k8s.io/depth=0" \
    -l "eng.tree.hnc.x-k8s.io/depth=1"
  namespace::check_exists new-prj \
    -a "configmanagement.gke.io/managed=enabled" \
    -a "hnc.x-k8s.io/managedBy=configmanagement.gke.io" \
    -a "hnc.x-k8s.io/managed-by=configmanagement.gke.io" \
    -l "new-prj.tree.hnc.x-k8s.io/depth=0" \
    -l "rnd.tree.hnc.x-k8s.io/depth=1"
  namespace::check_exists newer-prj \
    -a "configmanagement.gke.io/managed=enabled" \
    -a "hnc.x-k8s.io/managedBy=configmanagement.gke.io" \
    -a "hnc.x-k8s.io/managed-by=configmanagement.gke.io" \
    -l "newer-prj.tree.hnc.x-k8s.io/depth=0" \
    -l "rnd.tree.hnc.x-k8s.io/depth=1"
  resource::check_count -a "configmanagement.gke.io/managed=enabled" -r clusterrole -c 3
  resource::check_count -a "hnc.x-k8s.io/managedBy=configmanagement.gke.io" -r clusterrole -c 0
  resource::check_count -a "hnc.x-k8s.io/managed-by=configmanagement.gke.io" -r clusterrole -c 0
  resource::check clusterrole acme-admin -a "configmanagement.gke.io/managed=enabled"
  resource::check clusterrole namespace-viewer -a "configmanagement.gke.io/managed=enabled"
  resource::check clusterrole rbac-viewer -a "configmanagement.gke.io/managed=enabled"
  resource::check_count -a "configmanagement.gke.io/managed=enabled" -r clusterrolebinding -c 2
  resource::check_count -a "hnc.x-k8s.io/managedBy=configmanagement.gke.io" -r clusterrolebinding -c 0
  resource::check_count -a "hnc.x-k8s.io/managed-by=configmanagement.gke.io" -r clusterrolebinding -c 0
  resource::check clusterrolebinding namespace-viewers -a "configmanagement.gke.io/managed=enabled"
  resource::check clusterrolebinding rbac-viewers -a "configmanagement.gke.io/managed=enabled"
  resource::check_count -a "hnc.x-k8s.io/managedBy=configmanagement.gke.io" -r podsecuritypolicy -c 0
  resource::check_count -a "hnc.x-k8s.io/managed-by=configmanagement.gke.io" -r podsecuritypolicy -c 0
  resource::check podsecuritypolicy example -a "configmanagement.gke.io/managed=enabled"

  # Namespace-scoped resources
  # analytics
  resource::check_count -n analytics -r role -c 0
  resource::check_count -n analytics -r rolebinding -c 2
  resource::check_count -n analytics -r rolebinding -c 0 -a "hnc.x-k8s.io/managedBy=configmanagement.gke.io"
  resource::check_count -n analytics -r rolebinding -c 0 -a "hnc.x-k8s.io/managed-by=configmanagement.gke.io"
  resource::check -n analytics rolebinding mike-rolebinding -a "configmanagement.gke.io/managed=enabled"
  resource::check -n analytics rolebinding alice-rolebinding -a "configmanagement.gke.io/managed=enabled"
  resource::check_count -n analytics -r resourcequota -c 1 -a "configmanagement.gke.io/managed=enabled"
  resource::check_count -n analytics -r resourcequota -c 0 -a "hnc.x-k8s.io/managedBy=configmanagement.gke.io"
  resource::check_count -n analytics -r resourcequota -c 0 -a "hnc.x-k8s.io/managed-by=configmanagement.gke.io"
  resource::check -n analytics resourcequota pod-quota -a "configmanagement.gke.io/managed=enabled"

  # backend
  resource::check_count -n backend -r configmap -c 1 -l "app.kubernetes.io/managed-by=configmanagement.gke.io"
  resource::check -n backend configmap store-inventory -a "configmanagement.gke.io/managed=enabled"
  resource::check_count -n backend -r role -c 0
  resource::check_count -n backend -r rolebinding -c 2
  resource::check_count -n backend -r rolebinding -c 0 -a "hnc.x-k8s.io/managedBy=configmanagement.gke.io"
  resource::check_count -n backend -r rolebinding -c 0 -a "hnc.x-k8s.io/managed-by=configmanagement.gke.io"
  resource::check -n backend rolebinding bob-rolebinding -a "configmanagement.gke.io/managed=enabled"
  resource::check -n backend rolebinding alice-rolebinding -a "configmanagement.gke.io/managed=enabled"
  resource::check_count -n backend -r resourcequota -c 1 -a "configmanagement.gke.io/managed=enabled"
  resource::check_count -n backend -r resourcequota -c 0 -a "hnc.x-k8s.io/managedBy=configmanagement.gke.io"
  resource::check_count -n backend -r resourcequota -c 0 -a "hnc.x-k8s.io/managed-by=configmanagement.gke.io"
  resource::check -n backend resourcequota pod-quota -a "configmanagement.gke.io/managed=enabled"
  run kubectl get resourcequota -n backend -o yaml
  assert::contains 'pods: "1"'

  # frontend
  namespace::check_exists frontend -l "env=prod" -a "audit=true"
  resource::check_count -n frontend -r configmap -c 1 -l "app.kubernetes.io/managed-by=configmanagement.gke.io"
  resource::check -n frontend configmap store-inventory -a "configmanagement.gke.io/managed=enabled"
  resource::check_count -n frontend -r role -c 0
  resource::check_count -n frontend -r rolebinding -c 2
  resource::check_count -n frontend -r rolebinding -c 0 -a "hnc.x-k8s.io/managedBy=configmanagement.gke.io"
  resource::check_count -n frontend -r rolebinding -c 0 -a "hnc.x-k8s.io/managed-by=configmanagement.gke.io"
  resource::check -n frontend rolebinding alice-rolebinding -a "configmanagement.gke.io/managed=enabled"
  resource::check -n frontend rolebinding sre-admin -a "configmanagement.gke.io/managed=enabled"
  resource::check_count -n frontend -r resourcequota -c 1 -a "configmanagement.gke.io/managed=enabled"
  resource::check_count -n frontend -r resourcequota -c 0 -a "hnc.x-k8s.io/managedBy=configmanagement.gke.io"
  resource::check_count -n frontend -r resourcequota -c 0 -a "hnc.x-k8s.io/managed-by=configmanagement.gke.io"
  resource::check -n frontend resourcequota pod-quota -a "configmanagement.gke.io/managed=enabled"

  # new-prj
  resource::check_count -n new-prj -r role -c 1
  resource::check_count -n new-prj -r role -c 0 -a "hnc.x-k8s.io/managedBy=configmanagement.gke.io"
  resource::check_count -n new-prj -r role -c 0 -a "hnc.x-k8s.io/managed-by=configmanagement.gke.io"
  resource::check -n new-prj role acme-admin -a "configmanagement.gke.io/managed=enabled"
  resource::check_count -n new-prj -r rolebinding -c 0
  resource::check_count -n new-prj -r resourcequota -c 1 -a "configmanagement.gke.io/managed=enabled"
  resource::check_count -n new-prj -r resourcequota -c 0 -a "hnc.x-k8s.io/managedBy=configmanagement.gke.io"
  resource::check_count -n new-prj -r resourcequota -c 0 -a "hnc.x-k8s.io/managed-by=configmanagement.gke.io"
  resource::check -n new-prj resourcequota quota -a "configmanagement.gke.io/managed=enabled"

  # newer-prj
  resource::check_count -n newer-prj -r role -c 0
  resource::check_count -n newer-prj -r rolebinding -c 0
  resource::check_count -n newer-prj -r resourcequota -c 1 -a "configmanagement.gke.io/managed=enabled"
  resource::check_count -n newer-prj -r resourcequota -c 0 -a "hnc.x-k8s.io/managedBy=configmanagement.gke.io"
  resource::check_count -n newer-prj -r resourcequota -c 0 -a "hnc.x-k8s.io/managed-by=configmanagement.gke.io"
  resource::check -n newer-prj resourcequota quota -a "configmanagement.gke.io/managed=enabled"

  local services=(
    git-importer
    syncer
  )
  for service in "${services[@]}"; do
    check_metrics_pages ${service}
  done
}

