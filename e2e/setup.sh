#!/bin/bash

set -euo pipefail

TEST_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

FWD_SSH_PORT=2222

function set_up_kube() {
  readonly kubeconfig_output="/opt/installer/kubeconfig/config"
  # We need to fix up the kubeconfig paths because these may not match between
  # the container and the host.
  # /somepath/gcloud becomes /use/local/gcloud/google-cloud/sdk/bin/gcloud.
  # Then, make it read-writable to the owner only.
  cat /home/user/.kube/config | \
    sed -e "s+cmd-path: [^ ]*gcloud+cmd-path: /usr/local/gcloud/google-cloud-sdk/bin/gcloud+g" \
    > "${kubeconfig_output}"
  chmod 600 ${kubeconfig_output}
}

function set_up_env() {
  echo "++++ Setting up environment"
  case ${importer} in
    git)
    /opt/testing/e2e/init-git-server.sh
    ;;
    gcp)
    gsutil cp ${gcp_cred} /tmp
    ;;
  esac

  echo "+++++ Installing..."
  suggested_user="$(gcloud config get-value account)"
  /opt/installer/installer \
    --config="${install_config}" \
    --log_dir=/tmp \
    --suggested_user="${suggested_user}" \
    --use_current_context=true \
    --vmodule=main=10,configgen=10,kubectl=10,installer=10,exec=10
}

function set_up_env_minimal() {
  echo "++++ Setting up environment (minimal)"
  if [[ "$importer" == "git" ]]; then
    echo "Starting port forwarding"
    TEST_LOG_REPO=/tmp/nomos-test
    POD_ID=$(kubectl get pods -n=nomos-system-test -l app=test-git-server -o jsonpath='{.items[0].metadata.name}')
    mkdir -p ${TEST_LOG_REPO}
    kubectl -n=nomos-system-test port-forward ${POD_ID} ${FWD_SSH_PORT}:22 > ${TEST_LOG_REPO}/port-forward.log &
    local start=$(date +%s)
    local pid=$(pgrep kubectl)
    # Our image doesn't have netstat, so we have check for listen by
    # looking at kubectl's open tcp sockets in procfs.  This looks for listen on
    # 127.0.0.1:$FWD_SSH_PORT with remote of 0.0.0.0:0.
    # TODO: This should use git ls-remote, but for some reason it fails to be able
    # to connect to the remote.
    while ! grep "0100007F:08AE 00000000:0000 0A" /proc/${pid}/net/tcp &> /dev/null; do
      echo -n "."
      sleep 0.1
      if [[ $(( $(date +%s) - $start )) > 10 ]]; then
        echo "Failed to set up kubectl tunnel!"
        return 1
      fi
    done
  fi
}

function clean_up() {
  if ! $clean; then
    return 0
  fi

  echo "++++ Cleaning up environment"
  suggested_user="$(gcloud config get-value account)"
  echo "Uninstalling..."
  /opt/installer/installer \
    --config="${install_config}" \
    --log_dir=/tmp \
    --suggested_user="${suggested_user}" \
    --use_current_context=true \
    --uninstall=deletedeletedelete \
    --vmodule=main=10,configgen=10,kubectl=10,installer=10,exec=10

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

function main() {
  local filter="${1:-}"
  local file_filter=""
  local testcase_filter=""

  if [[ "$filter" == */* ]]; then
    file_filter="$(echo $filter | sed -e 's|/.*||')"
    testcase_filter="$(echo $filter | sed -e 's|[^/]*/||')"
    echo "Filtering for files matching: ${file_filter}"
  else
    testcase_filter="$filter"
  fi
  if [[ "${testcase_filter}" != "" ]]; then
    echo "Filtering for testcases matching: ${testcase_filter}"
  fi

  start_time=$(date +%s)
  if [[ ! "kubectl get ns > /dev/null" ]]; then
    echo "Kubectl/Cluster misconfigured"
    exit 1
  fi
  GIT_SSH_COMMAND="ssh -q -o StrictHostKeyChecking=no -i /opt/testing/e2e/id_rsa.nomos"; export GIT_SSH_COMMAND

  echo "++++ Starting tests"
  local bats_tests=()
  # We don't have any GCP tests yet, skip all tests.
  if [[ "$importer" == gcp ]]; then
    bats_tests=()
  else
    bats_tests=$(
      echo ${TEST_DIR}/e2e.bats;
      find ${TEST_DIR}/testcases -name '*.bats'
    )
  fi

  local testcases=()
  if [[ -n ${file_filter} ]]; then
    for file in ${bats_tests[@]}; do
      if echo "${file}" | grep "${file_filter}" &> /dev/null; then
        echo "Will run ${file}"
        testcases+=("${file}")
      fi
    done
  else
    if [[ "${#bats_tests[@]}" != "0" ]]; then
      for file in ${bats_tests[@]}; do
        testcases+=("${file}")
      done
    fi
  fi

  local bats_flags=()
  if $tap; then
    bats_flags+=(--tap)
  fi

  local retcode=0
  if [[ "${#testcases[@]}" != 0 ]]; then
    if ! E2E_TEST_FILTER="${testcase_filter}" ${TEST_DIR}/bats/bin/bats ${bats_flags[@]} ${testcases[@]}; then
      retcode=1
    fi
  else
    echo "No files to test!"
  fi

  end_time=$(date +%s)
  echo "Tests took $(( ${end_time} - ${start_time} )) seconds."
  return ${retcode}
}

echo "executed with args $@"
filter=""
tap=false
clean=false
run_tests=false
setup=false
gcp_cred=""
while [[ $# -gt 0 ]]; do
  arg=${1}
  case ${arg} in
    --tap)
      tap=true
      shift
    ;;

    --clean)
      clean=true
      shift
    ;;

    --setup)
      setup=true
      shift
    ;;

    --test)
      run_tests=true
      shift
    ;;

    --filter)
      filter="${2}"
      shift
      shift
    ;;
    --importer)
      importer="${2}"
      shift
      shift
    ;;
    --gcp-cred)
      gcp_cred="${2}"
      shift
      shift
    ;;
    *)
    shift
    ;;
  esac
done

install_config=""
case ${importer} in
  git)
  install_config="${TEST_DIR}/install-config-git.yaml"
  ;;
  gcp)
  install_config="${TEST_DIR}/install-config-gcp.yaml"
  ;;
  *)
   echo "invalid importer value: ${importer}"
   exit 1
  ;;
esac

set_up_kube
clean_up
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
trap clean_up EXIT

if $run_tests; then
  main ${filter}
else
  echo "Skipping tests!"
fi
