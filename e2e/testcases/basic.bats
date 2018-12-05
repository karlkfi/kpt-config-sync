#!/bin/bash

set -euo pipefail

YAML_DIR=${BATS_TEST_DIRNAME}/../testdata

load ../lib/loader

@test "Namespace garbage collection" {
  git::add ${YAML_DIR}/accounting-namespace.yaml acme/namespaces/eng/accounting/namespace.yaml
  git::commit
  wait::for kubectl get ns accounting
  git::rm acme/namespaces/eng/accounting/namespace.yaml
  git::commit
  wait::for -f -t 20 -- kubectl get ns accounting
  run kubectl get policynodes new-ns
  [ "$status" -eq 1 ]
  assert::contains "not found"
}

@test "Namespace to Policyspace conversion" {
  git::rm acme/namespaces/rnd/newer-prj/namespace.yaml
  git::add ${YAML_DIR}/accounting-namespace.yaml acme/namespaces/rnd/newer-prj/accounting/namespace.yaml
  git::commit

  wait::for kubectl get ns accounting
  kubectl get ns newer-prj
}

@test "RoleBindings updated" {
  run kubectl get rolebindings -n backend bob-rolebinding -o yaml
  assert::contains "acme-admin"

  git::update ${YAML_DIR}/robert-rolebinding.yaml acme/namespaces/eng/backend/bob-rolebinding.yaml
  git::commit
  wait::for -t 30 -f -- kubectl get rolebindings -n backend bob-rolebinding

  run kubectl get rolebindings -n backend bob-rolebinding
  assert::contains "NotFound"
  run kubectl get rolebindings -n backend robert-rolebinding -o yaml
  assert::contains "acme-admin"

  # verify that importToken has been updated from the commit above
  local itoken="$(kubectl get policynode backend -ojsonpath='{.spec.importToken}')"
  git::check_hash "$itoken"

  # verify that syncToken has been updated as well
  wait::for -t 30 -- policynode::sync_token_eq backend "$itoken"
}

@test "RoleBindings enforced" {
  run kubectl get pods -n backend --as bob@acme.com
  assert::contains "No resources"
  run kubectl get pods -n backend --as alice@acme.com
  assert::contains "No resources"

  run kubectl get pods -n frontend --as bob@acme.com
  assert::contains "forbidden"
  run kubectl get pods -n frontend --as alice@acme.com
  assert::contains "No resources"
}

@test "ResourceQuota enforced" {
  clean_test_configmaps
  wait::for kubectl get ns new-prj
  run kubectl create configmap map1 -n new-prj
  assert::contains "created"
  run kubectl create configmap map2 -n newer-prj
  assert::contains "created"
  run kubectl create configmap map3 -n new-prj
  assert::contains "exceeded quota in policyspace rnd"
  clean_test_configmaps
}

@test "ResourceQuota live uninstall/reinstall" {
  # Verify the resourcequota admission controller is currently installed
  wait::for kubectl get deployment -n nomos-system resourcequota-admission-controller
  wait::for kubectl get validatingwebhookconfigurations resource-quota.nomos.dev

  # Verify that disabling the resource quota admission controller causes it to be uninstalled
  kubectl apply -f ${BATS_TEST_DIRNAME}/../operator-config-git-no-rq.yaml
  wait::for -f -- kubectl get deployment -n nomos-system resourcequota-admission-controller
  wait::for -f -- kubectl get validatingwebhookconfigurations resource-quota.nomos.dev

  # Verify that re-enabling the resource quota admission controller causes it to be reinstalled
  kubectl apply -f ${BATS_TEST_DIRNAME}/../operator-config-git.yaml
  wait::for kubectl get deployment -n nomos-system resourcequota-admission-controller
  wait::for kubectl get validatingwebhookconfigurations resource-quota.nomos.dev
}

function clean_test_configmaps() {
  kubectl delete configmaps -n new-prj --all > /dev/null
  kubectl delete configmaps -n newer-prj --all > /dev/null
  wait::for -f -- kubectl -n new-prj configmaps map1
  wait::for -f -- kubectl -n newer-prj configmaps map2
}
