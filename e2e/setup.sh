#!/bin/bash

set -euo pipefail

readonly TEST_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

readonly FWD_SSH_PORT=2222

source ./e2e/lib/wait.bash

NOMOS_REPO="${NOMOS_REPO:-.}"

function nomos_running() {
  if ! kubectl get pods -n nomos-system | grep git-policy-importer | grep Running; then
    echo "Importer not yet running"
    return 1
  fi
  if ! kubectl get pods -n nomos-system | grep syncer | grep Running; then
    echo "Syncer not yet running"
    return 1
  fi
  return 0
}

function nomos_uninstalled() {
  if [ "$(kubectl get pods -n nomos-system | wc -l)" -eq 0 ]; then
    echo "Nomos pods not yet uninstalled"
    return 0
  fi
  return 1
}

# Runs the installer process to set up the cluster under test.
function install() {
  if $do_installation; then
    echo "+++++ Installing..."
    (
       case ${importer} in
        git)
        # Linter says this is better than "cd -"
        cd "${NOMOS_REPO}/.output/e2e"
        # This will fail in subsequent runs, but is necessary for the first run
        kubectl create clusterrolebinding cluster-admin-binding --clusterrole cluster-admin --user "$(gcloud config get-value account)" || true
        kubectl apply -f defined-operator-bundle.yaml
        kubectl create secret generic git-creds -n=nomos-system --from-file=ssh="$HOME"/.ssh/id_rsa.nomos
        kubectl create -f "${TEST_DIR}/operator-config-git.yaml"
        wait::for -s -t 60 -- nomos_running
        ;;

        gcp)
        # gcp is still using the old-style installer. This can be removed once this is not the case
        # Linter says this is better than "cd -"
        cd "${NOMOS_REPO}/.output/e2e/installer"
        "${run_installer}" \
          --config="${install_config}" \
          --work_dir="${PWD}"
        ;;

        *)
         echo "invalid importer value: ${importer}"
         exit 1
        ;;
      esac
    )
  fi
}

# Runs the uninstaller process to uninstall nomos in the cluster under test.
function uninstall() {
  if $do_installation; then
    # If we did the installation, then we should uninstall as well.
    echo "+++++ Uninstalling..."
    (
      case ${importer} in
        git)
        # we do not care if part of the deletion fails, as the objects may not all exist
        kubectl -n=nomos-system delete nomos --all || true
        wait::for -s -t 60 -- nomos_uninstalled
        kubectl delete -f defined-operator-bundle.yaml || true
        ;;

        gcp)
        # gcp is still using the old-style installer. This can be removed once this is not the case
        cd "${NOMOS_REPO}/.output/e2e/installer"
        "${run_installer}" \
          --config="${install_config}" \
          --work_dir="${PWD}" \
          --uninstall=deletedeletedelete
        ;;

        *)
         echo "invalid importer value: ${importer}"
         exit 1
        ;;
      esac
      # make sure that nomos-system is no longer extant
      kubectl delete ns nomos-system || true
      echo clean
    )
  fi
}


function set_up_env() {
  echo "++++ Setting up environment: ${importer}"
  case ${importer} in
    git)
    /opt/testing/e2e/init-git-server.sh
    ;;
    gcp)
    gsutil cp "${gcp_watcher_cred}" "$HOME"
    gsutil cp "${gcp_runner_cred}" "$HOME"
    ;;
  esac

  install
}

function set_up_env_minimal() {
  echo "++++ Setting up environment (minimal)"
  case ${importer} in
    git)
    echo "Starting port forwarding"
    TEST_LOG_REPO=/tmp/nomos-test
    POD_ID=$(kubectl get pods -n=nomos-system-test -l app=test-git-server -o jsonpath='{.items[0].metadata.name}')
    mkdir -p ${TEST_LOG_REPO}
    kubectl -n=nomos-system-test port-forward "${POD_ID}" "${FWD_SSH_PORT}:22" > ${TEST_LOG_REPO}/port-forward.log &
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
    ;;
    gcp)
    gsutil cp "${gcp_runner_cred}" "$HOME"
    ;;
  esac
}

function clean_up_test_resources() {
  kubectl delete --ignore-not-found ns -l "nomos.dev/testdata=true"
  kubectl delete --ignore-not-found ns -l "nomos.dev/namespace-management=full"

  if [[ "$importer" == "git" ]]; then
    echo "killing kubectl port forward..."
    pkill -f "kubectl -n=nomos-system-test port-forward.*${FWD_SSH_PORT}:22" || true
    echo "  taking down nomos-system-test namespace"
    kubectl delete --ignore-not-found ns nomos-system-test
    while kubectl get ns nomos-system-test > /dev/null 2>&1
    do
      sleep 3
      echo -n "."
    done
  fi
}

function clean_up() {

  echo "++++ Cleaning up environment"

  # TODO: workaround for b/111218567 remove this once resolved
  if ! kubectl get customresourcedefinition policynodes.nomos.dev &> /dev/null; then
    echo "Policynodes not found, skipping uninstall"
    clean_up_test_resources
    return
  fi

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
  if [[ "$importer" == gcp ]]; then
    mapfile -t all_test_files < <(find "${TEST_DIR}/gcp_testcases" -name '*.bats' | sort)
  else
    mapfile -t all_test_files < <(find "${TEST_DIR}/testcases" -name '*.bats' | sort)
  fi

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

  export IMPORTER="${importer}"

  if [[ "${testcase_filter}" != "" ]]; then
    export E2E_TEST_FILTER="${testcase_filter}"
  fi

  # prow test runner defines the $ARTIFACTS env variable pointing to a
  # directory to be used to output test results that are automatically
  # presented to various dashboards.
  local result_file=""
  local has_artifacts=false
  if [ -n "${ARTIFACTS+x}" ]; then
    result_file="${ARTIFACTS}/result_${importer}.bats"
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
      tap2junit --name "${importer}_tests" "${result_file}"
      # Testgrid requires this particular file name, and tap2junit doesn't allow
      # renames. So...
      mv "${result_file}".xml "${ARTIFACTS}/junit_${importer}.xml"
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

# Creates a public-private ssh key pair.
# TODO(filmil): Eliminate the need for the keypair to exist both in $HOME
# and /opt/testing.
create_keypair() {
  echo "+++ Creating keypair at: ${HOME}/.ssh"
  mkdir -p "$HOME/.ssh"
  # Skipping confirmation in keygen returns nonzero code even if it was a
  # success.
  (yes | ssh-keygen \
        -t rsa -b 4096 \
        -C "your_email@example.com" \
        -N '' -f "/opt/testing/e2e/id_rsa.nomos") || echo "Key created here."
  ln -s "/opt/testing/e2e/id_rsa.nomos" "${HOME}/.ssh/id_rsa.nomos"
}

echo "e2e/setup.sh: executed with args" "$@"

clean=false
file_filter=".*"
gcp_runner_cred=""
gcp_watcher_cred=""
importer=""
preclean=false
run_tests=false
setup=false
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

    --importer)
      importer="${1}"
      shift
    ;;
    --gcp-watcher-cred)
      gcp_watcher_cred="${1}"
      shift
    ;;
    --gcp-runner-cred)
      gcp_runner_cred="${1}"
      shift
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
  create_keypair
fi

install_config="${TEST_DIR}/install-config.yaml"
install_config_template=""
case ${importer} in
  git)
  install_config_template="${TEST_DIR}/install-config-git.yaml"
  ;;

  gcp)
  install_config_template="${TEST_DIR}/install-config-gcp.yaml"
  ;;

  *)
   echo "invalid importer value: ${importer}"
   exit 1
  ;;
esac

if $do_installation; then
  suggested_user="$(gcloud config get-value account)"
  kubectl_context="$(kubectl config current-context)"
  run_installer="${TEST_DIR}/installer/install.sh"
  sed -e "s/CONTEXT/${kubectl_context}/" -e "s/USER/${suggested_user}/" \
    < "${install_config_template}" \
    > "${install_config}"
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
