#!/bin/bash
#
# This testcase exists to verify that the "acme" org is created fully.
#

set -euo pipefail

load ../lib/loader

@test "All acme corp created" {
  local ns

  resource::check_count -r namespace -l nomos.dev/namespace-management -c ${#ACME_NAMESPACES[@]}

  namespace::check_exists analytics \
    -l "nomos.dev/parent-name=eng" \
    -l "nomos.dev/namespace-management=full"
  namespace::check_exists backend \
    -l "nomos.dev/parent-name=eng" \
    -l "nomos.dev/namespace-management=full"
  namespace::check_exists frontend \
    -l "nomos.dev/parent-name=eng" \
    -l "nomos.dev/namespace-management=full"
  namespace::check_exists new-prj \
    -l "nomos.dev/parent-name=rnd" \
    -l "nomos.dev/namespace-management=full"
  namespace::check_exists newer-prj \
    -l "nomos.dev/parent-name=rnd" \
    -l "nomos.dev/namespace-management=full"

  resource::check_count -r validatingwebhookconfigurations -c 2
  resource::check validatingwebhookconfigurations policy.nomos.dev
  resource::check validatingwebhookconfigurations resource-quota.nomos.dev

  resource::check_count -l "nomos.dev/managed=full" -r clusterrole -c 3
  resource::check clusterrole acme-admin -l "nomos.dev/managed=full"
  resource::check clusterrole namespace-viewer -l "nomos.dev/managed=full"
  resource::check clusterrole rbac-viewer -l "nomos.dev/managed=full"

  resource::check_count -l "nomos.dev/managed=full" -r clusterrolebinding -c 2
  resource::check clusterrolebinding namespace-viewers -l "nomos.dev/managed=full"
  resource::check clusterrolebinding rbac-viewers -l "nomos.dev/managed=full"

  resource::check_count -r podsecuritypolicy -c 1
  resource::check podsecuritypolicy example -l "nomos.dev/managed=full"

  resource::check_count -n analytics -r role -c 0
  resource::check_count -n analytics -r rolebinding -c 2
  resource::check -n analytics rolebinding analytics.mike-rolebinding -l "nomos.dev/managed=full"
  resource::check -n analytics rolebinding eng.alice-rolebinding -l "nomos.dev/managed=full"
  resource::check_count -n analytics -r resourcequota -c 1
  resource::check -n analytics resourcequota nomos-resource-quota -l "nomos.dev/managed=full"

  resource::check_count -n backend -r role -c 0
  resource::check_count -n backend -r rolebinding -c 2
  resource::check -n backend rolebinding backend.bob-rolebinding -l "nomos.dev/managed=full"
  resource::check -n backend rolebinding eng.alice-rolebinding -l "nomos.dev/managed=full"
  resource::check_count -n backend -r resourcequota -c 1
  resource::check -n backend resourcequota nomos-resource-quota -l "nomos.dev/managed=full"
  run kubectl get quota -n backend -o yaml
  assert::contains 'pods: "1"'

  namespace::check_exists frontend -l "env=prod" -a "audit=true"
  resource::check_count -n frontend -r role -c 0
  resource::check_count -n frontend -r rolebinding -c 1
  resource::check -n frontend rolebinding eng.alice-rolebinding -l "nomos.dev/managed=full"
  resource::check_count -n frontend -r resourcequota -c 1
  resource::check -n frontend resourcequota nomos-resource-quota -l "nomos.dev/managed=full"

  resource::check_count -n new-prj -r role -c 1
  resource::check -n new-prj role acme-admin -l "nomos.dev/managed=full"
  resource::check_count -n new-prj -r rolebinding -c 0
  resource::check_count -n new-prj -r resourcequota -c 1
  resource::check -n new-prj resourcequota nomos-resource-quota -l "nomos.dev/managed=full"

  resource::check_count -n newer-prj -r role -c 0
  resource::check_count -n newer-prj -r rolebinding -c 0
  resource::check_count -n newer-prj -r resourcequota -c 1
  resource::check -n newer-prj resourcequota nomos-resource-quota -l "nomos.dev/managed=full"
}

