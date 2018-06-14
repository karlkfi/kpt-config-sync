
TEST_REPO_DIR=${BATS_TMPDIR}

setup() {
  if [[ "${E2E_TEST_FILTER}" != "" ]]; then
    local cur_test=""
    for func in "${FUNCNAME[@]}"; do
      if echo "$func" | grep "^test_" &> /dev/null; then
        cur_test="${func}"
      fi
    done
    if ! [[ "${cur_test}" =~ "${E2E_TEST_FILTER}" ]]; then
      skip
    fi
  fi

  # Reset git repo to initial state.
  CWD=$(pwd)
  rm -rf ${TEST_REPO_DIR}/repo
  mkdir -p ${TEST_REPO_DIR}/repo
  cd ${TEST_REPO_DIR}/repo

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
  cd $CWD
  # Wait for syncer to update each object type.
  wait::for_success "kubectl get ns backend"
  wait::for_success "kubectl get rolebindings -n backend"
  wait::for_success "kubectl get roles -n new-prj"
  wait::for_success "kubectl get quota -n backend"
  # We delete bob-rolebinding in one test case, make sure it's restored.
  wait::for_success "kubectl get rolebindings bob-rolebinding -n backend"
}

teardown() {
  if type local_teardown &> /dev/null; then
    echo "Running local_teardown"
    local_teardown
  fi
}
