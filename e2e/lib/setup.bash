
TEST_REPO_DIR=${BATS_TMPDIR}

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

teardown() {
  if type local_teardown &> /dev/null; then
    echo "Running local_teardown"
    local_teardown
  fi
}
