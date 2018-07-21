#!/bin/bash

set -euo pipefail

YAML_DIR=${BATS_TEST_DIRNAME}/../testdata

load ../lib/loader

@test "Namespace garbage collection" {
  git::add ${YAML_DIR}/accounting-namespace.yaml acme/eng/accounting/namespace.yaml
  git::commit
  wait::for kubectl get ns accounting
  git::rm acme/eng/accounting/namespace.yaml
  git::commit
  wait::for -f -t 20 -- kubectl get ns accounting
  run kubectl get policynodes new-ns
  [ "$status" -eq 1 ]
  assert::contains "not found"
}

@test "Namespace to Policyspace conversion" {
  git::rm acme/rnd/newer-prj/namespace.yaml
  git::add ${YAML_DIR}/accounting-namespace.yaml acme/rnd/newer-prj/accounting/namespace.yaml
  git::commit

  wait::for kubectl get ns accounting
  kubectl get ns newer-prj
}

@test "RoleBindings updated" {
  run kubectl get rolebindings -n backend bob-rolebinding -o yaml
  assert::contains "acme-admin"

  git::update ${YAML_DIR}/robert-rolebinding.yaml acme/eng/backend/bob-rolebinding.yaml
  git::commit
  wait::for -f -- kubectl get rolebindings -n backend bob-rolebinding
  run kubectl get rolebindings -n backend bob-rolebinding
  assert::contains "NotFound"
  run kubectl get rolebindings -n backend robert-rolebinding -o yaml
  assert::contains "acme-admin"

  # verify that importToken has been updated from the commit above
  local itoken="$(kubectl get policynode backend -ojsonpath='{.spec.importToken}')"
  git::check_hash "$itoken"

  # sleep to try to reduce potential flakiness between syncer updating the resources and syncer
  # updating the syncToken of the policynodes
  # TODO(ekitson): Use wait::event when we publish a sync event to look for
  sleep 1

  # verify that syncToken has been updated as well
  run kubectl get policynode backend -ojsonpath='{.status.syncTokens.backend}'
  assert::equals "$itoken"

  # verify the other syncTokens in the hierarchy
  itoken="$(kubectl get policynode eng -ojsonpath='{.spec.importToken}')"
  run kubectl get policynode backend -ojsonpath='{.status.syncTokens.eng}'
  assert::equals "$itoken"
  itoken="$(kubectl get policynode acme -ojsonpath='{.spec.importToken}')"
  run kubectl get policynode backend -ojsonpath='{.status.syncTokens.acme}'
  assert::equals "$itoken"
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

function clean_test_configmaps() {
  kubectl delete configmaps -n new-prj --all > /dev/null
  kubectl delete configmaps -n newer-prj --all > /dev/null
  wait::for -f -- kubectl -n new-prj configmaps map1
  wait::for -f -- kubectl -n newer-prj configmaps map2
}
