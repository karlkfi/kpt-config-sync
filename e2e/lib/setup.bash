#!/bin/bash

set -e

TEST_REPO_DIR=${BATS_TMPDIR}

# Total count of namespaces in acme
ACME_NAMESPACES=(
  analytics
  backend
  frontend
  new-prj
  newer-prj
)

# System namespaces (kubernetes + nomos)
SYS_NAMESPACES=(
  default
  kube-public
  kube-system
  nomos-system
  nomos-system-test
)

setup() {
  E2E_TEST_FILTER="${E2E_TEST_FILTER:-}"
  if [[ "${E2E_TEST_FILTER}" != "" ]]; then
    local cur_test=""
    for func in "${FUNCNAME[@]}"; do
      if echo "$func" | grep "^test_" &> /dev/null; then
        cur_test="${func}"
      fi
    done
    if ! echo "${cur_test}" | grep "${E2E_TEST_FILTER}" &> /dev/null; then
      skip
    fi
  fi

  # Delete testdata that might exist.
  kubectl delete ns -l "nomos.dev/testdata=true" &> /dev/null || true

  # Reset git repo to initial state.
  CWD="$(pwd)"
  echo "Setting up local git repo"
  rm -rf "${TEST_REPO_DIR}/repo"
  mkdir -p "${TEST_REPO_DIR}/repo"
  cd "${TEST_REPO_DIR}/repo"

  git init
  git remote add origin ssh://git@localhost:2222/git-server/repos/sot.git
  git fetch
  git config user.name "Testing Nome"
  git config user.email testing_nome@example.com
  mkdir acme
  touch acme/README.md
  git add acme/README.md
  git commit -a -m "initial commit"

  cp -r /opt/testing/e2e/sot/acme ./
  git add -A
  git commit -m "setUp commit"
  git push origin master:master -f
  cd "$CWD"

  setup::wait_for_namespaces

  # Wait for syncer to update each object type.
  wait::for_success "kubectl get rolebindings -n backend"
  wait::for_success "kubectl get roles -n new-prj"
  wait::for_success "kubectl get quota -n backend"
  # We delete bob-rolebinding in one test case, make sure it's restored.
  wait::for_success "kubectl get rolebindings backend.bob-rolebinding -n backend"
  wait::for_success "kubectl get clusterrole acme-admin"
}

# Previous tests can create / delete namespaces. This will wait for the
# namespaces to finish terminating and the state to get restored to base acme.
function setup::wait_for_namespaces() {
  debug::log -n "Waiting for namespaces to stabilize"
  local start
  start=$(date +%s)
  local deadline
  deadline=$(( $(date +%s) + 30 ))
  while (( "$(date +%s)" < deadline )); do
    if setup::check_stable; then
      debug::log "Namespaces stabilized in $(( $(date +%s) - start )) seconds"
      return 0
    fi
    sleep 0.1
  done
  debug::log "Namespaces failed to stabilize to acme defaults, got:"
  debug::log "$(kubectl get ns)"
  return 1
}

function setup::check_stable() {
  debug::log "checking for stable"
  local ns_states
  ns_states="$(
    kubectl get ns -ojsonpath="{.items[*].status.phase}" \
    | tr ' ' '\n' \
    | sort \
    | uniq -c
  )"
  if echo "${ns_states}" | grep "Terminating" &> /dev/null; then
    local count
    count=$(echo "${ns_states}" | grep "Terminating" | sed -e 's/^ *//' -e 's/T.*//')
    debug::log "Waiting for $count namespaces to finalize"
    return 1
  fi

  local ns_count
  ns_count=$(resource::count -r ns -l nomos.dev/namespace-management)
  if (( ns_count != ${#ACME_NAMESPACES[@]} )); then
    debug::log "count mismatch $ns_count != ${#ACME_NAMESPACES[@]}"
    return 1
  fi

  echo "Checking namespaces for active state"
  for ns in "${ACME_NAMESPACES[@]}" "${SYS_NAMESPACES[@]}"; do
    if ! kubectl get ns "${ns}" -oyaml | grep "phase: Active" &> /dev/null; then
      debug::log "namespace ${ns} not active yet"
      return 1
    fi
  done
}

teardown() {
  if type local_teardown &> /dev/null; then
    echo "Running local_teardown"
    local_teardown
  fi
}

