#!/bin/bash

set -euo pipefail

readonly TEST_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

readonly FWD_SSH_PORT=2222

function set_up_env() {
  echo "++++ Setting up environment"
  case ${importer} in
    git)
    /opt/testing/e2e/init-git-server.sh
    ;;
    gcp)
    gsutil cp "${gcp_watcher_cred}" "$HOME"
    gsutil cp "${gcp_runner_cred}" "$HOME"
    ;;
  esac

  echo "+++++ Installing..."
  cd "${NOMOS_REPO}/.output/e2e"
  "${run_installer}" --config="$(basename "${install_config}")" \
    --container "$CONTAINER" \
    --version "$VERSION"
  cd -
}

function set_up_env_minimal() {
  echo "++++ Setting up environment (minimal)"
  if [[ "$importer" == "git" ]]; then
    echo "Starting port forwarding"
    TEST_LOG_REPO=/tmp/nomos-test
    POD_ID=$(kubectl get pods -n=nomos-system-test -l app=test-git-server -o jsonpath='{.items[0].metadata.name}')
    mkdir -p ${TEST_LOG_REPO}
    kubectl -n=nomos-system-test port-forward "${POD_ID}" "${FWD_SSH_PORT}:22" > ${TEST_LOG_REPO}/port-forward.log &
    local pid=$!
    local start_time
    start_time=$(date +%s)
    # Our image doesn't have netstat, so we have check for listen by
    # looking at kubectl's open tcp sockets in procfs.  This looks for listen on
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
  fi
}

function clean_up() {

  echo "++++ Cleaning up environment"

  # TODO: workaround for b/111218567 remove this once resolved
  if ! kubectl get customresourcedefinition policynodes.nomos.dev &> /dev/null; then
    echo "Policynodes not found, skipping uninstall"
    return
  fi

  cd "${NOMOS_REPO}/.output/e2e"
  "${run_installer}" --config="$(basename "${install_config}")" \
    --uninstall=deletedeletedelete \
    --container "$CONTAINER" \
    --version "$VERSION"
  cd -

  kubectl delete ns -l "nomos.dev/testdata=true"
  kubectl delete ns -l "nomos.dev/namespace-management=full"

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

  echo "++++ Starting tests"
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
        echo "Will run ${file}"
        filtered_test_files+=("${file}")
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

  local retcode=0
  if (( ${#filtered_test_files[@]} != 0 )); then
    if ! "${bats_cmd[@]}" "${filtered_test_files[@]}"; then
      retcode=1
    fi
  else
    echo "No files to test!"
  fi

  end_time=$(date +%s)
  echo "Tests took $(( end_time - start_time )) seconds."
  return ${retcode}
}

echo "executed with args" "$@"
file_filter=".*"
tap=false
preclean=false
clean=false
run_tests=false
setup=false
gcp_watcher_cred=""
gcp_runner_cred=""
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
    *)
      echo "Unrecognized arg $arg"
      exit 1
    ;;
  esac
done

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

export PATH="${GCLOUD_PATH}:${KUBECTL_PATH}:$PATH"
export KUBECONFIG=${HOME}/.kube/config
suggested_user="$(gcloud config get-value account)"
kubectl_context="$(kubectl config current-context)"
run_installer="${TEST_DIR}/run-installer.sh"
sed -e "s/CONTEXT/${kubectl_context}/" -e "s/USER/${suggested_user}/" \
  < "${install_config_template}" \
  > "${install_config}"

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
