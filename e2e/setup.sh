#!/bin/bash

echo "Start setup.sh"

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
  name: ${account}
EOF
}

function ensure_config_management_removed() {
  echo "++++++ Removing ConfigManagement CRD"
  if ! kubectl delete crd configmanagements.configmanagement.gke.io --ignore-not-found --timeout=30s; then
    echo "++++++ Timed out waiting for ConfigManagement CRD to be deleted normally b/138222737"
    # We enter this block if both:
    # 1) The ConfigManagement CRD exists, and
    # 2) Deleting that CRD timed out.

    # Since the normal deletion process failed, then the operator is not running properly and
    # we have to manually remove the finalizer, then apply the ConfigManagement without the
    # problematic finalizer. We assume this is the standard ConfigManagement named
    # "config-management", and that the problem preventing deletion is the finalizer.
    kubectl get configmanagements config-management -oyaml >> "${TEST_DIR}/tmp-config-management.yaml"
    grep -vwE "operator.configmanagement.gke.io" "${TEST_DIR}/tmp-config-management.yaml" >> "${TEST_DIR}/tmp-config-management-without-finalizer.yaml"
    kubectl apply -f "${TEST_DIR}/tmp-config-management-without-finalizer.yaml"
    if ! kubectl delete crd configmanagements.configmanagement.gke.io --ignore-not-found --timeout=30s; then
      echo "++++++ Exiting Because Unable to Remove ConfigManagement CRD"
      exit 1
    fi
  fi
  echo "++++++ Removed ConfigManagement CRD"
}

# Runs the installer process to set up the cluster under test.
function install() {
  if ${do_installation}; then
    echo "+++++ Installing"
    # Make sure config-management-system doesn't exist before installing.
    # The google3/ move shouldn't require this as clusters will not persist between tests.
    ensure_config_management_removed

    kubectl apply -f "${TEST_DIR}/defined-operator-bundle.yaml"
    kubectl create secret generic git-creds -n=config-management-system \
      --from-file=ssh="${TEST_DIR}/id_rsa.nomos" || true
    if ${stable_channel}; then
      echo "++++++ Using Nomos stable channel"
      MANIFEST_DIR="${TEST_DIR}/manifests/stable"
    else
      echo "++++++ Using Nomos dev channel"
      MANIFEST_DIR="${TEST_DIR}/manifests/dev"
    fi

    export MANIFEST_DIR

    echo "++++++ Applying Nomos Initial Configuration"
    kubectl apply -f "${MANIFEST_DIR}/operator-config-git.yaml"

    echo "++++++ Waiting for config-management-system deployments to be up"
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
  if ${do_installation}; then
    # If we did the installation, then we should uninstall as well.
    echo "+++++ Uninstalling"

    if kubectl get configmanagement &> /dev/null; then
      ensure_config_management_removed
    fi
    echo "++++++ Wait to confirm shutdown"
    wait::for -s -t 300 -- install::nomos_uninstalled
    echo "++++++ Delete operator bundle"
    kubectl delete --ignore-not-found -f defined-operator-bundle.yaml --timeout=30s

    # make sure that config-management-system no longer exists.
    if ! kubectl get ns config-management-system &> /dev/null; then
      echo "Error: config-management-system was not deleted during operator removal."
    fi
    echo clean
  fi
}

function set_up_env() {
  echo "++++ Setting up environment"
  if ${gotopt2_set_admin_role_binding:-false}; then
    apply_cluster_admin_binding "$(gcloud config get-value account)"
  fi

  # NOTE: we have to set up the git server first, otherwise there's a race
  # condition where the importer can come up faster than the git server and have
  # to wait two minutes for timeout.
  echo "++++ Setting up git server"
  /opt/testing/e2e/init-git-server.sh
  echo "++++ Setting up Nomos"
  install
  echo "++++ Env setup complete"
}

function set_up_env_minimal() {
  echo "++++ Setting up environment (minimal)"

  echo "Starting port forwarding"
  TEST_LOG_REPO=/tmp/nomos-test
  POD_ID=$(kubectl get pods -n=config-management-system-test -l app=test-git-server -o jsonpath='{.items[0].metadata.name}')
  mkdir -p ${TEST_LOG_REPO}
  kubectl -n=config-management-system-test port-forward "${POD_ID}" \
    "${FWD_SSH_PORT}:22" > ${TEST_LOG_REPO}/port-forward.log &
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
  echo "printing diagnostics"
  echo "+ operator logs"
  (kubectl -n kube-system logs -l k8s-app=config-management-operator --tail=100) || true
  echo "+ importer pod"
  (kubectl -n config-management-system describe pod git-importer) || true
  echo "+ importer logs"
  (kubectl -n config-management-system logs -l app=git-importer -c importer --tail=100) || true

  kubectl delete --ignore-not-found ns -l "configmanagement.gke.io/testdata=true"
  resource::delete -r ns -a configmanagement.gke.io/managed=enabled

  echo "killing kubectl port forward..."
  pkill -f "kubectl -n=config-management-system-test port-forward.*${FWD_SSH_PORT}:22" || true
  echo "  taking down config-management-system-test namespace"
  kubectl delete --ignore-not-found ns config-management-system-test
  wait::for -f -t 100 -- kubectl get ns config-management-system-test
}

function clean_up() {
  echo "++++ Cleaning up environment"

  declare -a waitpids
  uninstall & waitpids+=("${!}")
  clean_up_test_resources & waitpids+=("${!}")
  echo "++++ Waiting for environment cleanup to finish"
  wait "${waitpids[@]}"
}

function post_clean() {
  if ${clean}; then
    clean_up
  fi
}

function main() {
  local file_filter="${1}"
  local testcase_filter=""

  start_time=$(date +%s)
  if ! kubectl get ns > /dev/null; then
    echo "Kubectl/Cluster misconfigured"
    exit 1
  fi
  GIT_SSH_COMMAND="ssh -q -o StrictHostKeyChecking=no -i /opt/testing/e2e/id_rsa.nomos"; export GIT_SSH_COMMAND

  # TODO(fmil): remove the root reason for this message.
  # shellcheck disable=SC2154
  echo "+++ Starting tests from ${gotopt2_testcases_dir}"
  local all_test_files=()
  mapfile -t all_test_files < <(find "${TEST_DIR}/${gotopt2_testcases_dir}" -name '*.bats' | sort)

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
  if ${tap}; then
    bats_cmd+=(--tap)
  fi

  if [[ "${testcase_filter}" != "" ]]; then
    export E2E_TEST_FILTER="${testcase_filter}"
  fi

  if ${timing}; then
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
  # shellcheck disable=SC2154
  gcloud --quiet container clusters get-credentials "${gcp_cluster_name}" \
    --zone="${gotopt2_zone}" --project="${gotopt2_project}"
}

echo "e2e/setup.sh: executed with args" "$@"

# gotopt2 binary is built into the e2e container.
readonly gotopt2_result=$(/opt/gotopt2 "${@}" << EOF
flags:
- name: "tap"
  type: bool
  help: "If set, produces test output in TAP format"
- name: "preclean"
  type: bool
  help: "If set, executes the 'preclean' step of the setup"
- name: "clean"
  type: bool
  help: "If set, executes the 'clean' step of the setup"
- name: "setup"
  type: bool
  help: "If set, executes the 'setup' step of the setup - installs Nomos"
- name: "test"
  type: bool
  help: "If set, executes the 'test' step of the setup - runs tests"
- name: "timing"
  type: bool
  help: "If set, prints test timing in the testing output"
- name: "stable_channel"
  type: bool
  help: "If set, uses stable operator channel for testing"
- name: "test_filter"
  type: string
  help: "A regex defining the test names of tests to execute"
- name: "file_filter"
  type: string
  help: "A regex defining the file names of tests to execute"
  default: ".*"
- name: "gcp-prober-cred"
  type: string
  help: "If set, the supplied credentials file will be used to set up the identity of the test runner. Otherwise defaults to whatever user is the current default in gcloud and kubectl config."
- name: "gcp-cluster-name"
  type: string
  help: "The name of the GCP cluster to use for testing.  This is used to obtain cluster credentials at the start of the installation process"
  default: "${USER}-cluster-1"
- name: "skip_installation"
  type: bool
  help: "If set, skips the installation step"
- name: "create-ssh-key"
  type: bool
  help: "If set, the setup will create a new ssh key to use in the tests"
  default: false
- name: "set-admin-role-binding"
  type: bool
  help: "If set, the setup will add the admin role binding"
  default: true
- name: "testcases-dir"
  type: string
  default: "testcases"
  help: "The directory name, relative to testdir to pick e2e tests up from"
- name: "zone"
  type: string
  default: "us-central1-a"
  help: "The zone in which the test cluster is situated"
- name: "project"
  type: string
  default: "stolos-dev"
  help: "The project in which the test cluster was created"
EOF
)
eval "${gotopt2_result}"

readonly tap="${gotopt2_tap:-false}"
readonly preclean="${gotopt2_preclean:-false}"
readonly clean="${gotopt2_clean:-false}"
readonly setup="${gotopt2_setup:-false}"
readonly timing="${gotopt2_timing:-false}"
readonly stable_channel="${gotopt2_stable_channel:-false}"
# TODO(filmil): remove the need to disable lint checks here and elsewhere.
# shellcheck disable=SC2154
readonly test_filter="${gotopt2_test_filter}"
export E2E_TEST_FILTER="${test_filter}"
readonly skip_installation="${gotopt2_skip_installation:-false}"
readonly create_ssh_key="${gotopt2_create_ssh_key:-false}"
# shellcheck disable=SC2154
readonly gcp_cluster_name="${gotopt2_gcp_cluster_name}"
# shellcheck disable=SC2154
readonly gcp_prober_cred="${gotopt2_gcp_prober_cred}"
readonly run_tests="${gotopt2_test:-false}"

do_installation=true
if [[ "${skip_installation}" == "true" ]]; then
  do_installation=true
fi

if [[ "${gotopt2_gcp_prober_cred}" != "" ]]; then
  setup_prober_cred "${gotopt2_gcp_prober_cred}"
fi


if ${create_ssh_key}; then
  install::create_keypair
fi

if ${preclean}; then
  clean_up
fi

echo "++++ kubectl version"
kubectl version

if ${setup}; then
  set_up_env
else
  if ${clean}; then
    echo "Already cleaned up, skipping minimal setup!"
  else
    set_up_env_minimal
  fi
fi

# Always run clean_up before exit at this point
trap post_clean EXIT

if ${run_tests}; then
  # shellcheck disable=SC2154
  main "${gotopt2_file_filter}"
else
  echo "Skipping tests!"
fi
