#!/bin/bash

set -euo pipefail

readonly TEST_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

readonly FWD_SSH_PORT=2222

# shellcheck source=e2e/lib/wait.bash
source "$TEST_DIR/lib/wait.bash"
# shellcheck source=e2e/lib/install.bash
source "$TEST_DIR/lib/install.bash"
# shellcheck source=e2e/lib/resource.bash
source "$TEST_DIR/lib/resource.bash"

function apply_cluster_admin_binding() {
  local account="${1:-}"
  kubectl apply -f - <<EOF
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: cluster-admin-binding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluster-admin
subjects:
- apiGroup: rbac.authorization.k8s.io
  kind: User
  name: $account
EOF
}

# Runs the installer process to set up the cluster under test.
function install() {
  if $do_installation; then
    echo "+++++ Installing..."
    # Linter says this is better than "cd -"
    apply_cluster_admin_binding "$(gcloud config get-value account)"
    kubectl apply -f "${TEST_DIR}/defined-operator-bundle.yaml"
    kubectl create secret generic git-creds -n=config-management-system \
      --from-file=ssh="${TEST_DIR}/id_rsa.nomos" || true
    if $stable_channel; then
      echo "+++++ Applying Nomos using stable channel"
      kubectl apply -f "${TEST_DIR}/operator-config-git-stable.yaml"
    else
      echo "+++++ Applying Nomos using dev channel"
      kubectl apply -f "${TEST_DIR}/operator-config-git.yaml"
    fi
    echo "+++++   Waiting for config-management-system deployments to be up"
    wait::for -s -t 180 -- install::nomos_running

    local image
    image="$(kubectl get pods -n config-management-system \
      -l app=syncer \
      -ojsonpath='{.items[0].spec.containers[0].image}')"
    echo "Nomos $image up and running"
  fi
}

# Runs the uninstaller process to uninstall nomos in the cluster under test.
function uninstall() {
  if $do_installation; then
    # If we did the installation, then we should uninstall as well.
    echo "+++++ Uninstalling..."
    if kubectl get configmanagement &> /dev/null; then
      kubectl -n=config-management-system delete configmanagement --all
    fi
    wait::for -s -t 300 -- install::nomos_uninstalled
    kubectl delete --ignore-not-found -f defined-operator-bundle.yaml

    # make sure that config-management-system is no longer existant
    if kubectl delete ns config-management-system &> /dev/null; then
      echo "Error: config-management-system was not deleted during operator removal."
    fi
    echo clean
  fi
}

function set_up_env() {
  echo "++++ Setting up environment"
  /opt/testing/e2e/init-git-server.sh

  install
}

function set_up_env_minimal() {
  echo "++++ Setting up environment (minimal)"

  echo "Starting port forwarding"
  TEST_LOG_REPO=/tmp/nomos-test
  POD_ID=$(kubectl get pods -n=config-management-system-test -l app=test-git-server -o jsonpath='{.items[0].metadata.name}')
  mkdir -p ${TEST_LOG_REPO}
  kubectl -n=config-management-system-test port-forward "${POD_ID}" "${FWD_SSH_PORT}:22" > ${TEST_LOG_REPO}/port-forward.log &
  local pid=$!
  local start_time
  start_time=$(date +%s)
  # Our image doesn't have netstat, so we have check for listen by
  # looking at kubectl's Uninstalling tcp sockets in procfs.  This looks for listen on
  # 127.0.0.1:$FWD_SSH_PORT with remote of 0.0.0.0:0.
  # TODO: This should use git ls-remote, but for some reason it fails to be able
  # to connect to the remote.
  while ! grep "0100007F:08AE 00000000:0000 0A" "/proc/${pid}/net/tcp" &> /dev/null; do
    echo -n "."
    sleep 0.1
    if (( $(date +%s) - start_time > 10 )); then
      echo "Failed to set up kubectl tunnel!"
      return 1
    fi
  done
}

function clean_up_test_resources() {
  kubectl delete --ignore-not-found ns -l "configmanagement.gke.io/testdata=true"
  # TODO(125862145): Remove as part of rename cleanup
  resource::delete -r ns -a nomos.dev/managed=enabled
  resource::delete -r ns -a configmanagement.gke.io/managed=enabled

  # TODO: this is to work around the label to annotation switch, delete this
  # sometime after 2018-02-28
  local i
  for i in clusterrolebinding clusterrole podsecuritypolicy; do
    # TODO(125862145): Remove as part of rename cleanup
    resource::delete -r $i -a nomos.dev/managed=enabled
    resource::delete -r $i -a configmanagement.gke.io/managed=enabled
    kubectl delete -l nomos.dev/managed=enabled $i
  done

  echo "killing kubectl port forward..."
  pkill -f "kubectl -n=config-management-system-test port-forward.*${FWD_SSH_PORT}:22" || true
  echo "  taking down config-management-system-test namespace"
  kubectl delete --ignore-not-found ns config-management-system-test
  wait::for -f -t 100 -- kubectl get ns config-management-system-test
}

function clean_up() {

  echo "++++ Cleaning up environment"

  uninstall

  clean_up_test_resources
}

function post_clean() {
  if $clean; then
    clean_up
  fi
}

function main() {
  local file_filter="${1:-}"
  local testcase_filter=""

  start_time=$(date +%s)
  if ! kubectl get ns > /dev/null; then
    echo "Kubectl/Cluster misconfigured"
    exit 1
  fi
  GIT_SSH_COMMAND="ssh -q -o StrictHostKeyChecking=no -i /opt/testing/e2e/id_rsa.nomos"; export GIT_SSH_COMMAND

  echo "+++ Starting tests"
  local all_test_files=()
  mapfile -t all_test_files < <(find "${TEST_DIR}/testcases" -name '*.bats' | sort)

  local filtered_test_files=()
  if [[ "${file_filter}" == "" ]]; then
    file_filter=".*"
  fi

  if (( ${#all_test_files[@]} != 0 )); then
    for file in "${all_test_files[@]}"; do
      if echo "${file}" | grep "${file_filter}" &> /dev/null; then
        if [ -x "${file}" ]  && [ -r "${file}" ]; then
          echo "+++ Will run ${file}"
          filtered_test_files+=("${file}")
        else
          echo "### File not readable or executable: ${file}"
          stat "${file}"
          return 1
        fi
      fi
    done
  fi

  local bats_cmd=("${TEST_DIR}/bats/bin/bats")
  if $tap; then
    bats_cmd+=(--tap)
  fi

  if [[ "${testcase_filter}" != "" ]]; then
    export E2E_TEST_FILTER="${testcase_filter}"
  fi

  if [ -n "${timing+x}" ]; then
    export TIMING="${timing}"
  fi

  # prow test runner defines the $ARTIFACTS env variable pointing to a
  # directory to be used to output test results that are automatically
  # presented to various dashboards.
  local result_file=""
  local has_artifacts=false
  if [ -n "${ARTIFACTS+x}" ]; then
    result_file="${ARTIFACTS}/result_git.bats"
    has_artifacts=true
  fi

  local retcode=0
  if (( ${#filtered_test_files[@]} != 0 )); then
    if "${has_artifacts}"; then
      # TODO(filmil): Find a way to unify the 'if' and 'else' branches without
      # side effects on log output.
      echo "+++ Adding results also to: ${result_file}"
      if ! "${bats_cmd[@]}" "${filtered_test_files[@]}" | tee "${result_file}"; then
        retcode=1
      fi
    else
      if ! "${bats_cmd[@]}" "${filtered_test_files[@]}"; then
        retcode=1
      fi
    fi

    if "${has_artifacts}"; then
      echo "+++ Converting test results from TAP format to jUnit"
      /opt/tap2junit -reorder_duration -test_name="git_tests" \
        < "${result_file}" \
        > "${ARTIFACTS}/junit_git.xml"
    fi
  else
    echo "No files to test!"
  fi

  end_time=$(date +%s)
  echo "Tests took $(( end_time - start_time )) seconds."
  return ${retcode}
}

# Configures gcloud and kubectl to use supplied service account credentials
# to interact with the cluster under test.
#
# Params:
#   $1: File path to the JSON file containing service account credentials.
#   #2: Optional, GCS file that we want to download the configuration from.
setup_prober_cred() {
  echo "+++ Setting up prober credentials"
  local cred_file="$1"

  # Makes the service account from ${_cred_file} the active account that drives
  # cluster changes.
  gcloud --quiet auth activate-service-account --key-file="${cred_file}"

  # Installs gcloud as an auth helper for kubectl with the credentials that
  # were set with the service account activation above.
  # Needs cloud.containers.get permission.
  gcloud --quiet container clusters get-credentials \
    "${gcp_cluster_name}" --zone us-central1-a --project stolos-dev
}


echo "e2e/setup.sh: executed with args" "$@"

clean=false
file_filter=".*"
preclean=false
run_tests=false
setup=false
stable_channel=false
tap=false

# If set, the tests will skip the installation.
do_installation=true

# If set, the supplied credentials file will be used to set up the identity
# of the test runner.  Otherwise defaults to whatever user is the current
# default in gcloud and kubectl config.
gcp_prober_cred=""

# The name of the cluster to use for testing by default.    This is used to
# obtain cluster credentials at start of the installation process.
gcp_cluster_name="$USER-cluster-1"

# If set, the setup will create a new ssh key to use in the tests.
create_ssh_key=false

while [[ $# -gt 0 ]]; do
  arg=${1}
  shift
  case ${arg} in
    --tap)
      tap=true
    ;;

    --preclean)
      preclean=true
    ;;

    --clean)
      clean=true
    ;;

    --setup)
      setup=true
    ;;

    --test)
      run_tests=true
    ;;

    --timing)
      timing=true
      tap=true
    ;;

    --stable_channel)
      stable_channel=true
    ;;

    --test_filter)
      test_filter="${1}"
      export E2E_TEST_FILTER="${test_filter}"
      shift
    ;;
    --test_filter=*)
      test_filter="$(sed -e 's/[^=]*=//' <<< "$arg")"
      export E2E_TEST_FILTER="${test_filter}"
    ;;

    --file_filter)
      file_filter="${1}"
      shift
    ;;
    --file_filter=*)
      file_filter="$(sed -e 's/[^=]*=//' <<< "$arg")"
    ;;

    --gcp-prober-cred)
      gcp_prober_cred="${1}"
      shift
    ;;
    --gcp-cluster-name)
      gcp_cluster_name="${1}"
      shift
    ;;
    --skip-installation)
      do_installation=false
    ;;
    --create-ssh-key)
      create_ssh_key=true
    ;;
    *)
      echo "setup.sh: unrecognized arg $arg"
      exit 1
    ;;
  esac
done

if [[ "${gcp_prober_cred}" != "" ]]; then
  setup_prober_cred "${gcp_prober_cred}"
fi

if ${create_ssh_key}; then
  install::create_keypair
fi

if $preclean; then
  clean_up
fi

if $setup; then
  set_up_env
else
  if $clean; then
    echo "Already cleaned up, skipping minimal setup!"
  else
    set_up_env_minimal
  fi
fi

# Always run clean_up before exit at this point
trap post_clean EXIT

if $run_tests; then
  main "${file_filter}"
else
  echo "Skipping tests!"
fi
