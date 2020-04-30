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


setup::git::__init_dir() {
  local DIR_NAME=${1}

  local TEST_REPO_DIR=${BATS_TMPDIR}
  cd "${TEST_REPO_DIR}/repo"

  mkdir "${DIR_NAME}"
  touch "${DIR_NAME}/README.md"
  git add "${DIR_NAME}/README.md"
  git commit -a -m "initial commit"

  cp -r "/opt/testing/e2e/examples/${DIR_NAME}" ./
}

setup::git::__commit_dir_and_wait() {
  git add -A
  git status
  git commit -m "setUp commit"
  git push origin master:master -f
  cd "$CWD"

  wait::for -t 60 -- nomos::repo_synced
}

# Adds an example repo to git, pushes it, and blocks until the proper namespaces are created.
setup::git::init() {
  local DIR_NAME=${1}

  setup::git::__init_dir "${DIR_NAME}"
  setup::git::__commit_dir_and_wait
}

# Same as init, but allows for specified files/directories to be excluded from an example repo.
#
# Variadic Args:
#   The paths of any files/directories from the example repo to exclude
# Example:
# - To exclude directory foo/bar/ and file foo/bazz.yaml:
#   setup::git::init_without() foo -- foo/bar/ foo/bazz.yaml
setup::git::init_without() {
  if [[ $# -eq 0 ]]; then
    echo "Error: No paths supplied to 'init_without'. Use 'init' if there are no paths to exclude"
    exit 1
  fi

  if [[ $# -lt 3 ]]; then
    echo "Error: must supply both example directory and files to exclude, delineated by '--'.  Use 'init' if there are no paths to exclude"
    exit 1
  fi

  if [[ ! " $* " =~ " -- " ]]; then
    echo "Error: missing '--'.  Must delineate repo and files using like: exampleDir -- exampleDir/aFile exampleDir/anotherFile"
    exit 1
  fi

  if [[ $2 != "--" ]]; then
    echo "Error: misplaced '--'.  Proper format: exampleDir -- exampleDir/aFile exampleDir/anotherFile"
  fi

  local DIR_NAME=$1
  shift 2

  setup::git::__init_dir "${DIR_NAME}"
  for path in "$@"; do
    if [[ -d ${path} ]]; then
      rm -r ${path}
    elif [[ -f ${path} ]]; then
      rm $path
    elif ! [[ -e ${path} ]]; then
      echo "Error: Specified path $path does not exist in the specified repo"
      exit 1
    fi
  done
  setup::git::__commit_dir_and_wait
}

# Removes *almost* all managed resources of the specified repo from git, pushes it, and blocks until complete.
setup::git::remove_all() {
  local DIR_NAME=${1}

  local TEST_REPO_DIR=${BATS_TMPDIR}
  cd "${TEST_REPO_DIR}/repo/${DIR_NAME}"

  rm -rf "cluster"
  rm -rf "namespaces"

  namespace::declare safety

  git add -A
  git status
  git commit -m "wiping almost all of ${DIR_NAME}"
  git push origin master:master -f
  cd "$CWD"

  wait::for -t 60 -- nomos::repo_synced

  setup::git::remove_all_dangerously "${DIR_NAME}"
}

# Removes all managed resources of the specified repo from git, pushes it, and blocks until complete.
setup::git::remove_all_dangerously() {
  local DIR_NAME=${1}

  local TEST_REPO_DIR=${BATS_TMPDIR}
  cd "${TEST_REPO_DIR}/repo/${DIR_NAME}"

  rm -rf "cluster"
  rm -rf "namespaces"

  git add -A
  git status
  git commit -m "wiping contents of ${DIR_NAME}"
  git push origin master:master -f
  cd "$CWD"

  wait::for -t 60 -- nomos::repo_synced
}

# Adds the contents of the specified example directory to the git repository's root
#
# This is part of the tests that evaluate behavior with undefined POLICY_DIR
#
setup::git::add_contents_to_root() {
  local DIR_NAME=${1}

  local TEST_REPO_DIR=${BATS_TMPDIR}
  cd "${TEST_REPO_DIR}/repo"

  cp -r "/opt/testing/e2e/examples/${DIR_NAME}/"* ./

  git add -A
  git commit -m "add files to root"
  git push origin master:master -f
  cd "$CWD"

  wait::for -t 60 -- nomos::repo_synced
}

# This removes the specified folder, leaving behind the other files in the project root
# 
# This is part of the tests that evaluate behavior with undefined POLICY_DIR
#
setup::git::remove_folder() {
  local DIR_NAME=${1}

  local TEST_REPO_DIR=${BATS_TMPDIR}
  cd "${TEST_REPO_DIR}/repo"

  rm -r "./${DIR_NAME}"

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

# Record and print timing info. This is cheap, and all tests should do it.
setup::common_teardown() {
  local end
  end=$(date +%s%3N)
  ((runtime="$end - $START_TIME"))

  if [ -n "${TIMING:+1}" ]; then
    echo "# TAP2JUNIT: Duration: ${runtime}ms" >&3
  fi
}
