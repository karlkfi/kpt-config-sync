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
# shellcheck source=e2e/lib/namespaceconfig.bash
source "$DIR/namespaceconfig.bash"
# shellcheck source=e2e/lib/nomos.bash
source "$DIR/nomos.bash"
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
  config-management-system
  config-management-system-test
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


setup::git::__init_acme_dir() {
  local TEST_REPO_DIR=${BATS_TMPDIR}
  cd "${TEST_REPO_DIR}/repo"

  mkdir acme
  touch acme/README.md
  git add acme/README.md
  git commit -a -m "initial commit"

  cp -r /opt/testing/e2e/examples/acme ./
}

setup::git::__commit_acme_dir_and_wait() {
  git add -A
  git commit -m "setUp commit"
  git push origin master:master -f
  cd "$CWD"

  wait::for -t 60 -- nomos::repo_synced
}

# Adds the example acme repo to git, pushes it, and blocks until the proper namespaces are created.
setup::git::init_acme() {
  setup::git::__init_acme_dir
  setup::git::__commit_acme_dir_and_wait
}

# Same as init_acme, but allows for specified files/directories to be excluded from the acme repo.
#
# Variadic Args:
#   The paths of any files/directories from the acme example repo to exclude
# Example:
# - To exclude directory foo/bar/ and file foo/bazz.yaml:
#   setup::git::init_acme() foo/bar/ foo/bazz.yaml
setup::git::init_acme_without() {
  if [[ $# -eq 0 ]]; then
    echo "Error: No paths supplied to 'init_acme_without'. Use 'init_acme' if there are no paths to exclude"
    exit 1
  fi

  setup::git::__init_acme_dir
  for path in "$@"; do
    if [[ -d ${path} ]]; then
      rm -r ${path}
    elif [[ -f ${path} ]]; then
      rm $path
    elif ! [[ -e ${path} ]]; then
      echo "Error: Specified path $path does not exist in the acme repo"
      exit 1
    fi
  done
  setup::git::__commit_acme_dir_and_wait
}

# Adds the contents of the acme directory to the git repository's root
#
# This is part of the tests that evaluate behavior with undefined POLICY_DIR
#
setup::git::add_acme_contents_to_root() {
  local TEST_REPO_DIR=${BATS_TMPDIR}
  cd "${TEST_REPO_DIR}/repo"

  cp -r /opt/testing/e2e/examples/acme/* ./

  git add -A
  git commit -m "add files to root"
  git push origin master:master -f
  cd "$CWD"

  wait::for -t 60 -- nomos::repo_synced
}

# This removes the acme folder, leaving behind the other files in the project root
# 
# This is part of the tests that evaluate behavior with undefined POLICY_DIR
#
setup::git::remove_acme_folder() {
  local TEST_REPO_DIR=${BATS_TMPDIR}
  cd "${TEST_REPO_DIR}/repo"

  rm -r ./acme

  git add -A
  git commit -m "add files to root"
  git push origin master:master -f
  cd "$CWD"

  wait::for -t 60 -- nomos::repo_synced
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
  ns_count=$(resource::count -r ns -a configmanagement.gke.io/managed)
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
