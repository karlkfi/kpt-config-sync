#!/bin/bash

set -euo pipefail

load "../lib/assert"
load "../lib/git"
load "../lib/setup"
load "../lib/wait"

setup() {
  setup::common
  setup::git::initialize

  local TEST_REPO_DIR=${BATS_TMPDIR}
  cd "${TEST_REPO_DIR}/repo"

  mkdir -p acme/system
  cp -r /opt/testing/e2e/examples/acme/system acme
  git add -A
}

YAML_DIR=${BATS_TEST_DIRNAME}/../testdata

@test "Namespace garbage collection" {
  mkdir -p acme/namespaces/accounting
  git::add ${YAML_DIR}/accounting-namespace.yaml acme/namespaces/accounting/namespace.yaml
  git::commit
  wait::for kubectl get ns accounting
  git::rm acme/namespaces/accounting/namespace.yaml
  git::commit
  wait::for -f -t 60 -- kubectl get ns accounting
  run kubectl get policynodes new-ns
  [ "$status" -eq 1 ]
  assert::contains "not found"
}

@test "Namespace to Policyspace conversion" {
  git::add ${YAML_DIR}/dir-namespace.yaml acme/namespaces/dir/namespace.yaml
  git::commit
  wait::for kubectl get ns dir

  git::rm acme/namespaces/dir/namespace.yaml
  git::add ${YAML_DIR}/subdir-namespace.yaml acme/namespaces/dir/subdir/namespace.yaml
  git::commit

  wait::for kubectl get ns subdir
  wait::for -f -- kubectl get ns dir
}

@test "RoleBindings updated" {
  git::add /opt/testing/e2e/examples/acme/namespaces/eng/backend/namespace.yaml acme/namespaces/eng/backend/namespace.yaml
  git::add /opt/testing/e2e/examples/acme/namespaces/eng/backend/bob-rolebinding.yaml acme/namespaces/eng/backend/br.yaml
  git::commit
  wait::for -- kubectl get rolebinding -n backend bob-rolebinding

  run kubectl get rolebindings -n backend bob-rolebinding -o yaml
  assert::contains "acme-admin"

  git::update ${YAML_DIR}/robert-rolebinding.yaml acme/namespaces/eng/backend/br.yaml
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
  git::add /opt/testing/e2e/examples/acme/namespaces/eng/backend/namespace.yaml acme/namespaces/eng/backend/namespace.yaml
  git::add /opt/testing/e2e/examples/acme/namespaces/eng/backend/bob-rolebinding.yaml acme/namespaces/eng/backend/br.yaml
  git::add /opt/testing/e2e/examples/acme/namespaces/eng/alice-rolebinding.yaml acme/namespaces/eng/backend/ar.yaml
  git::add /opt/testing/e2e/examples/acme/namespaces/eng/frontend/namespace.yaml acme/namespaces/eng/frontend/namespace.yaml
  git::commit
  wait::for -- kubectl get rolebinding -n backend bob-rolebinding
  wait::for -- kubectl get rolebinding -n backend alice-rolebinding

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
  git::add /opt/testing/e2e/examples/acme/namespaces/rnd acme/namespaces/rnd/
  git::commit
  wait::for kubectl get ns new-prj
  wait::for kubectl get ns newer-prj
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
