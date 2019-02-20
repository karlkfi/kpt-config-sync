#!/bin/bash

set -euo pipefail

DIR=$(dirname "${BASH_SOURCE[0]}")
# shellcheck source=e2e/lib/debug.bash
source "$DIR/debug.bash"
# shellcheck source=e2e/lib/git.bash
source "$DIR/git.bash"
# shellcheck source=e2e/lib/ignore.bash
source "$DIR/ignore.bash"
# shellcheck source=e2e/lib/namespace.bash
source "$DIR/namespace.bash"
# shellcheck source=e2e/lib/policynode.bash
source "$DIR/policynode.bash"
# shellcheck source=e2e/lib/resource.bash
source "$DIR/resource.bash"
# shellcheck source=e2e/lib/wait.bash
source "$DIR/wait.bash"

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

# Git-specific repository initialization.
setup::git::initialize() {
  # Reset git repo to initial state.
  CWD="$(pwd)"
  echo "Setting up local git repo"

  local TEST_REPO_DIR=${BATS_TMPDIR}
  rm -rf "${TEST_REPO_DIR}/repo"
  mkdir -p "${TEST_REPO_DIR}/repo"
  cd "${TEST_REPO_DIR}/repo"

  git init
  git remote add origin ssh://git@localhost:2222/git-server/repos/sot.git
  git fetch
  git config user.name "Testing Nome"
  git config user.email testing_nome@example.com
}

setup::git::init_acme() {
  local TEST_REPO_DIR=${BATS_TMPDIR}
  cd "${TEST_REPO_DIR}/repo"

  mkdir acme
  touch acme/README.md
  git add acme/README.md
  git commit -a -m "initial commit"

  cp -r /opt/testing/e2e/examples/acme ./
  git add -A
  git commit -m "setUp commit"
  git push origin master:master -f
  cd "$CWD"

  local ns
  local commit_hash
  commit_hash="$(git::hash)"
  for ns in "${ACME_NAMESPACES[@]}"; do
    wait::for -- policynode::sync_token_eq "${ns}" "${commit_hash}"
  done
}

# Record timing info, and implement the test filter. This is cheap, and all tests should do it.
setup::common() {
  START_TIME=$(date +%s%3N)

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

  echo "--- SETUP COMPLETE ---------------------------------------------------"
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
  ns_count=$(resource::count -r ns -a nomos.dev/managed)
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

# Record and print timing info. This is cheap, and all tests should do it.
setup::common_teardown() {
  local end
  end=$(date +%s%3N)
  ((runtime="$end - $START_TIME"))

  if [ -n "${TIMING:+1}" ]; then
    echo "# TAP2JUNIT: Duration: ${runtime}ms" >&3
  fi
}
