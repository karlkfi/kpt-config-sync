
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

# Total count of namespaces on system
TOTAL_NS_COUNT=$(( ${#ACME_NAMESPACES[@]} + ${#SYS_NAMESPACES[@]} ))


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
  echo "Setting up local git repo"
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
  wait::for_success "kubectl get rolebindings backend.bob-rolebinding -n backend"

  # Note that we have to wait for "Terminating" namespaces from previous
  # testcases to finalize so this can actually take quite some time.
  echo -n "Waiting for namespaces to stabilize"
  for i in $(seq 1 30); do
    if setup::check_stable; then
      echo
      echo "Namespaces stabilized"
      return 0
    fi
    echo -n "."
    sleep 0.25
  done
  echo
  echo "Namespaces failed to stabilize to acme defaults, got:"
  echo "$(kubectl get ns)"
  false
}

function setup::check_stable() {
  debug::log "checking for stable"
  local ns_count=$(resource::count -r ns)
  if [[ $ns_count != ${TOTAL_NS_COUNT} ]]; then
    debug::log "count mismatch $ns_count != ${TOTAL_NS_COUNT}"
    return 1
  fi

  echo "checking namespaces for active state"
  for ns in "${ACME_NAMESPACES[@]}" "${SYS_NAMESPACES[@]}"; do
    if ! kubectl get ns ${ns} -oyaml | grep "phase: Active"; then
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

