#!/bin/bash

TEST_REPO_DIR=${BATS_TMPDIR}

# Run for every test
setup() {
  CWD=$(pwd)
  cd ${TEST_REPO_DIR}
  rm -rf repo
  git clone ssh://git@localhost:2222/git-server/repos/sot.git ${TEST_REPO_DIR}/repo
  cd ${TEST_REPO_DIR}/repo
  git config user.name "Testing Nome"
  git config user.email testing_nome@example.com
  git rm -qrf acme
  cp -r /opt/testing/sot/acme ./
  git add -A
  git diff-index --quiet HEAD || git commit -m "setUp commit"
  git push origin master
  cd $CWD
  # Wait for syncer to update objects following the policynode updates
  wait_for_success "kubectl get ns backend"
  wait_for_success "kubectl get ns frontend"
}

# assertContains <command> <substring>
# Will fail if the output of the command or its error message doesn't contain substring
function assertContains() {
  if [[ "$output" != *"$1"* ]]; then
    echo "FAIL: [$output] does not contain [$1]"
    false
  fi
}

load e2e-git-api

######################## TESTS ########################
@test "syncer namespace" {
  run kubectl get ns eng
  assertContains "NotFound"
  run kubectl get ns backend
  assertContains "Active"
  run kubectl get ns frontend
  assertContains "Active"

  run kubectl get ns frontend -o yaml
  assertContains 'nomos.dev/namespace-management: full'
  run kubectl get ns frontend -o yaml
  assertContains "nomos-parent-ns: eng"
}

@test "syncer roles" {
  run kubectl get roles -n new-prj
  assertContains "acme-admin"
}

@test "syncer role bindings" {
  run kubectl get rolebindings -n backend bob-rolebinding -o yaml
  assertContains "acme-admin"

  run kubectl get rolebindings -n backend -o yaml
  assertContains "alice"
  run kubectl get rolebindings -n frontend -o yaml
  assertContains "alice"
}

@test "syncer role bindings change" {
  run kubectl get rolebindings -n backend bob-rolebinding -o yaml
  assertContains "acme-admin"
  git_update ${BATS_TEST_DIRNAME}/test-syncer-change-rolebinding-backend.yaml acme/eng/backend/bob-rolebinding.yaml
  git_commit
  wait_for_failure "kubectl get rolebindings -n backend bob-rolebinding"
  run kubectl get rolebindings -n backend bob-rolebinding
  assertContains "NotFound"
  run kubectl get rolebindings -n backend robert-rolebinding -o yaml
  assertContains "acme-admin"
}

@test "syncer quota" {
  run kubectl get quota -n backend -o yaml
  assertContains 'pods: "1"'
}

@test "authorizer" {
  skip "broken currently"
  run kubectl get pods -n backend --as bob@acme.com
  assertContains "No resources"
  run kubectl get pods -n backend --as alice@acme.com
  assertContains "No resources"

  run kubectl get pods -n frontend --as bob@acme.com
  assertContains "pods is forbidden"
}

@test "quota admission" {
  cleanTestConfigMaps
  wait_for_success "kubectl get ns new-prj"
  run kubectl create configmap map1 -n new-prj
  assertContains "created"
  run kubectl create configmap map2 -n newer-prj
  assertContains "created"
  run kubectl create configmap map3 -n new-prj
  assertContains "exceeded quota in policyspace rnd"
  cleanTestConfigMaps
}

@test "namespace garbage collection" {
  git_add $BATS_TEST_DIRNAME/testcases/yaml/namespace-garbage-collection-namespace.yaml acme/eng/test-syncer-namespaces/namespace.yaml
  git_commit
  wait_for_success "kubectl get ns test-syncer-namespaces"
  git_rm acme/eng/test-syncer-namespaces/namespace.yaml
  git_commit
  wait_for_failure "kubectl get ns test-syncer-namespaces" 20
  run kubectl get policynodes new-ns
  [ "$status" -eq 1 ]
  assertContains "not found"
}

function cleanTestConfigMaps() {
  kubectl delete configmaps -n new-prj --all > /dev/null
  kubectl delete configmaps -n newer-prj --all > /dev/null
  wait_for_failure "kubectl -n new-prj configmaps | grep map1"
  wait_for_failure "kubectl -n newer-prj configmaps | grep map2"
}


function wait_for_success() {
  local command="${1:-}"
  local timeout=${2:-10}
  local or_die=${3:-true}
  wait_for "${command}" "${timeout}" "${or_die}" true
}

function wait_for_failure() {
  local command="${1:-}"
  local timeout=${2:-10}
  local or_die=${3:-true}
  wait_for "${command}" "${timeout}" "${or_die}" false
}

function wait_for() {
  local command="${1:-}"
  local timeout=${2:-10}
  local or_die=${3:-true}
  local expect=${4:-true}

  echo -n "Waiting for ${command} to exit ${expect}"
  for i in $(seq 1 ${timeout}); do
    run ${command} &> /dev/null
    if [ $status -eq 0 ]; then
      if ${expect}; then
        echo
        return 0
      fi
    else
      if ! ${expect}; then
        echo
        return 0
      fi
    fi
    echo -n "."
    sleep 0.5
  done
  echo
  echo "Command '${command}' failed after ${timeout} seconds"
  if ${or_die}; then
    exit 1
  fi
}

