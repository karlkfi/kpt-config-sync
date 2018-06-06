#!/bin/bash

set -euo pipefail

TEST_REPO_DIR=${BATS_TMPDIR}
YAML_DIR=${BATS_TEST_DIRNAME}/yaml

load lib/assert
load lib/git
load lib/wait

# Runs for every test (Called by Bats framework).
setup() {
  CWD=$(pwd)
  cd ${TEST_REPO_DIR}
  rm -rf repo
  git clone ssh://git@localhost:2222/git-server/repos/sot.git ${TEST_REPO_DIR}/repo
  cd ${TEST_REPO_DIR}/repo
  git config user.name "Testing Nome"
  git config user.email testing_nome@example.com
  git rm -qrf acme
  cp -r /opt/testing/e2e/sot/acme ./
  git add -A
  git diff-index --quiet HEAD || git commit -m "setUp commit"
  git push origin master
  cd $CWD
  # Wait for syncer to update each object type.
  wait::for_success "kubectl get ns backend"
  wait::for_success "kubectl get rolebindings -n backend"
  wait::for_success "kubectl get roles -n new-prj"
  wait::for_success "kubectl get quota -n backend"
  # We delete bob-rolebinding in one test case, make sure it's restored.
  wait::for_success "kubectl get rolebindings bob-rolebinding -n backend"
}

@test "Namespaces created" {
  run kubectl get ns eng
  assert::contains "NotFound"
  run kubectl get ns backend
  assert::contains "Active"
  run kubectl get ns frontend
  assert::contains "Active"

  run kubectl get ns frontend -o yaml
  assert::contains 'nomos.dev/namespace-management: full'
  run kubectl get ns frontend -o yaml
  assert::contains "nomos-parent-ns: eng"
}

@test "Namespace garbage collection" {
  git::add ${YAML_DIR}/accounting-namespace.yaml acme/eng/accounting/namespace.yaml
  git::commit
  wait::for_success "kubectl get ns accounting"
  git::rm acme/eng/accounting/namespace.yaml
  git::commit
  wait::for_failure "kubectl get ns accounting" 20
  run kubectl get policynodes new-ns
  [ "$status" -eq 1 ]
  assert::contains "not found"
}

@test "Namespace to Policyspace conversion" {
  git::rm acme/rnd/newer-prj/namespace.yaml
  git::add ${YAML_DIR}/accounting-namespace.yaml acme/rnd/newer-prj/accounting/namespace.yaml
  git::commit

  wait::for_success "kubectl get ns accounting"
  kubectl get ns newer-prj
}

@test "Roles created" {
  run kubectl get roles -n new-prj
  assert::contains "acme-admin"
}

@test "RoleBindings created" {
  run kubectl get rolebindings -n backend bob-rolebinding -o yaml
  assert::contains "acme-admin"

  run kubectl get rolebindings -n backend -o yaml
  assert::contains "alice"
  run kubectl get rolebindings -n frontend -o yaml
  assert::contains "alice"
}

@test "RoleBindings updated" {
  run kubectl get rolebindings -n backend bob-rolebinding -o yaml
  assert::contains "acme-admin"
  git::update ${YAML_DIR}/robert-rolebinding.yaml acme/eng/backend/bob-rolebinding.yaml
  git::commit
  wait::for_failure "kubectl get rolebindings -n backend bob-rolebinding"
  run kubectl get rolebindings -n backend bob-rolebinding
  assert::contains "NotFound"
  run kubectl get rolebindings -n backend robert-rolebinding -o yaml
  assert::contains "acme-admin"
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

@test "ResourceQuota created" {
  run kubectl get quota -n backend -o yaml
  assert::contains 'pods: "1"'
}

@test "ResourceQuota enforced" {
  clean_test_configmaps
  wait::for_success "kubectl get ns new-prj"
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
  wait::for_failure "kubectl -n new-prj configmaps | grep map1"
  wait::for_failure "kubectl -n newer-prj configmaps | grep map2"
}
