#!/bin/bash

set -euo pipefail

load "../lib/assert"
load "../lib/git"
load "../lib/setup"
load "../lib/wait"
load "../lib/resource"

FILE_NAME="$(basename "${BATS_TEST_FILENAME}" '.bats')"

setup() {
  setup::common
  setup::git::initialize

  local TEST_REPO_DIR=${BATS_TMPDIR}
  cd "${TEST_REPO_DIR}/repo"

  mkdir -p acme/system
  cp -r /opt/testing/e2e/examples/acme/system acme
  git add -A
}

teardown() {
  setup::git::remove_all acme
  setup::common_teardown
}

YAML_DIR=${BATS_TEST_DIRNAME}/../testdata

@test "${FILE_NAME}: Recreate a sync deleted by the user" {
  debug::log "Add a deployment to a directory"
  git::add ${YAML_DIR}/dir-namespace.yaml acme/namespaces/dir/namespace.yaml
  git::add ${YAML_DIR}/deployment-helloworld.yaml acme/namespaces/dir/deployment.yaml
  git::commit -m "Add a deployment to a directory"

  debug::log "Ensure that the system created a sync for the deployment"
  wait::for -t 60 -- kubectl get sync deployment.apps

  debug::log "Force-delete the sync from the cluster"
  wait::for -t 60 -- kubectl delete sync deployment.apps

  debug::log "Ensure that the system re-created the force-deleted sync"
  wait::for -t 60 -- kubectl get sync deployment.apps
}

@test "${FILE_NAME}: Namespace garbage collection" {
  mkdir -p acme/namespaces/accounting
  git::add ${YAML_DIR}/accounting-namespace.yaml acme/namespaces/accounting/namespace.yaml
  git::commit
  wait::for kubectl get ns accounting
  git::rm acme/namespaces/accounting/namespace.yaml
  git::commit
  wait::for -f -t 60 -- kubectl get ns accounting
  run kubectl get namespaceconfigs new-ns
  [ "$status" -eq 1 ]
  assert::contains "not found"
}

@test "${FILE_NAME}: Namespace to Policyspace conversion" {
  git::add ${YAML_DIR}/dir-namespace.yaml acme/namespaces/dir/namespace.yaml
  git::commit
  wait::for kubectl get ns dir

  git::rm acme/namespaces/dir/namespace.yaml
  git::add ${YAML_DIR}/subdir-namespace.yaml acme/namespaces/dir/subdir/namespace.yaml
  git::commit

  wait::for kubectl get ns subdir
  wait::for -f -- kubectl get ns dir
}

@test "${FILE_NAME}: Start syncer and only sync namespace" {
  debug::log "Restarting syncer with an empty repo"
  kubectl delete pods -n config-management-system -l app=syncer
  wait::for -f -- kubectl get ns dir

  debug::log "Add only a single namespace"
  git::add ${YAML_DIR}/dir-namespace.yaml acme/namespaces/dir/namespace.yaml
  git::commit
  wait::for kubectl get ns dir
}

@test "${FILE_NAME}: Deleting deployment removes corresponding replicaset" {
  debug::log "Add a deployment"
  git::add ${YAML_DIR}/dir-namespace.yaml acme/namespaces/dir/namespace.yaml
  git::add ${YAML_DIR}/deployment-helloworld.yaml acme/namespaces/dir/deployment.yaml
  git::commit

  wait::for kubectl get ns dir

  debug::log "check that the deployment and replicaset were created"
  wait::for -t 60 -- kubectl get deployment hello-world -n dir
  wait::for -t 60 -- resource::check -r replicaset -n dir -l "app=hello-world"

  debug::log "Remove the deployment"
  git::rm acme/namespaces/dir/deployment.yaml
  git::commit

  debug::log "check that the deployment and replicaset were removed"
  wait::for -f -t 60 -- kubectl get deployment hello-world -n dir
  wait::for -f -t 60 -- resource::check replicaset -n dir -l "app=hello-world"
}

@test "${FILE_NAME}: Sync deployment and replicaset" {
  debug::log "Add a deployment and corresponding replicaset"
  git::add ${YAML_DIR}/dir-namespace.yaml acme/namespaces/dir/namespace.yaml
  git::add ${YAML_DIR}/deployment-helloworld.yaml acme/namespaces/dir/deployment.yaml
  git::add ${YAML_DIR}/replicaset-helloworld.yaml acme/namespaces/dir/replicaset.yaml
  git::commit

  debug::log "check that the deployment and replicaset were created"
  wait::for -t 60 -- kubectl get deployment hello-world -n dir
  wait::for -t 60 -- resource::check -r replicaset -n dir -l "app=hello-world"

  debug::log "Remove the deployment"
  git::rm acme/namespaces/dir/deployment.yaml
  git::commit

  debug::log "check that the deployment was removed and replicaset remains"
  wait::for -f -t 60 -- kubectl get deployment hello-world -n dir
  wait::for -t 60 -- resource::check -r replicaset -n dir -l "app=hello-world"
}

@test "${FILE_NAME}: RoleBindings updated" {
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

  # verify that import token has been updated from the commit above
  local itoken="$(kubectl get namespaceconfig backend -ojsonpath='{.spec.token}')"
  git::check_hash "$itoken"

  # verify that sync token has been updated as well
  wait::for -t 30 -- namespaceconfig::sync_token_eq backend "$itoken"
}

@test "${FILE_NAME}: RoleBindings enforced" {
  git::add /opt/testing/e2e/examples/acme/cluster/admin-clusterrole.yaml acme/cluster/admin-clusterrole.yaml
  git::add /opt/testing/e2e/examples/acme/namespaces/eng/backend/namespace.yaml acme/namespaces/eng/backend/namespace.yaml
  git::add /opt/testing/e2e/examples/acme/namespaces/eng/backend/bob-rolebinding.yaml acme/namespaces/eng/backend/br.yaml
  git::add /opt/testing/e2e/examples/acme/namespaces/eng/alice-rolebinding.yaml acme/namespaces/eng/ar.yaml
  git::add /opt/testing/e2e/examples/acme/namespaces/eng/frontend/namespace.yaml acme/namespaces/eng/frontend/namespace.yaml
  git::commit
  wait::for -- kubectl get rolebinding -n backend bob-rolebinding
  wait::for -- kubectl get rolebinding -n backend alice-rolebinding
  wait::for -- kubectl get clusterrole acme-admin

  debug::log "Verify that bob@ and alice@ have their permissions enforced"
  wait::for -o "No resources found." -t 60 -- \
    kubectl get pods -n backend --as bob@acme.com
  wait::for -o "No resources found." -t 60 -- \
    kubectl get pods -n backend --as alice@acme.com
  wait::for -f -t 60 -- \
    kubectl get pods -n frontend --as bob@acme.com
  wait::for -o "No resources found." -t 60 -- \
    kubectl get pods -n frontend --as alice@acme.com
}

@test "${FILE_NAME}: ResourceQuota enforced" {
  clean_test_configmaps
  git::add /opt/testing/e2e/examples/acme/namespaces/rnd acme/namespaces/rnd/
  git::commit

  wait::for -- kubectl get -n new-prj resourcequota config-management-resource-quota
  wait::for -- kubectl get -n newer-prj resourcequota config-management-resource-quota

  run kubectl create configmap map1 -n new-prj
  assert::contains "created"
  run kubectl create configmap map2 -n newer-prj
  assert::contains "created"
  run kubectl create configmap map3 -n new-prj
  assert::contains "exceeded quota in namespaces/rnd"
  clean_test_configmaps
}

function manage_namespace() {
  local ns="${1}"
  debug::log "Add an unmanaged resource into the namespace as a control"
  debug::log "We should never modify this resource"
  wait::for -t 5 -- \
    kubectl apply -f "${YAML_DIR}/reserved_namespaces/unmanaged-service.${ns}.yaml"

  debug::log "Add resource to manage"
  git::add "${YAML_DIR}/reserved_namespaces/service.yaml" \
    "acme/namespaces/${ns}/service.yaml"
  debug::log "Start managing the namespace"
  git::add "${YAML_DIR}/reserved_namespaces/namespace.${ns}.yaml" \
    "acme/namespaces/${ns}/namespace.yaml"
  git::commit -m "Start managing the namespace"

  debug::log "Wait until managed service appears on the cluster"
  wait::for -t 30 -- kubectl get services "some-service" --namespace="${ns}"

  debug::log "Remove the namespace directory from the repo"
  git::rm "acme/namespaces/${ns}"
  git::commit -m "Remove the namespace from the managed set of namespaces"

  debug::log "Wait until the managed resource disappears from the cluster"
  wait::for -f -t 60 -- kubectl get services "some-service" --namespace="${ns}"

  debug::log "Ensure that the unmanaged service remained"
  wait::for -t 60 -- \
    kubectl get services "some-other-service" --namespace="${ns}"
  kubectl delete service "some-other-service" \
    --ignore-not-found --namespace="${ns}"
}

@test "${FILE_NAME}: Namespace default can be managed" {
  manage_namespace "default"
}

@test "${FILE_NAME}: Namespace kube-system can be managed" {
  manage_namespace "kube-system"
}

@test "${FILE_NAME}: Namespace kube-public can be managed" {
  manage_namespace "kube-public"
}

@test "${FILE_NAME}: Namespace kube-whatever can be managed and has a normal lifecycle" {
  debug::log "Create a system namespace -- it is not required to exist"
  kubectl get ns "kube-whatever" || kubectl create ns "kube-whatever"
  manage_namespace "kube-whatever"
  debug::log "Ensure that the namespace is cleaned up at the end"
  wait::for -f -- kubectl get ns "kube-whatever"
}

function clean_test_configmaps() {
  kubectl delete configmaps -n new-prj --all > /dev/null
  kubectl delete configmaps -n newer-prj --all > /dev/null
  wait::for -f -- kubectl -n new-prj configmaps map1
  wait::for -f -- kubectl -n newer-prj configmaps map2
}

