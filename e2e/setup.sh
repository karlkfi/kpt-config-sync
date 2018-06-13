#!/bin/bash

set -euo pipefail

TEST_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

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
  echo "****************** Setting up environment ******************"
  /opt/testing/e2e/init-git-server.sh
  suggested_user="$(gcloud config get-value account)"
  /opt/installer/installer \
    --config="${TEST_DIR}/install-config.yaml" \
    --log_dir=/tmp \
    --suggested_user="${suggested_user}" \
    --use_current_context=true \
    --vmodule=main=10,configgen=10,kubectl=10,installer=10,exec=10
  echo "****************** Environment is ready ******************"
}

function set_up_env_minimal() {
  echo "****************** Minimal environment setup ******************"
  TEST_LOG_REPO=/tmp/nomos-test
  FWD_SSH_PORT=2222
  POD_ID=$(kubectl get pods -n=nomos-system -l app=test-git-server -o jsonpath='{.items[0].metadata.name}')
  mkdir -p ${TEST_LOG_REPO}
  kubectl -n=nomos-system port-forward ${POD_ID} ${FWD_SSH_PORT}:22 > ${TEST_LOG_REPO}/port-forward.log &
  kubectl exec -n=nomos-system -it ${POD_ID} -- git init --bare --shared /git-server/repos/sot.git
  echo "****************** Environment is ready ******************"
}

function clean_up() {
  echo "****************** Cleaning up environment ******************"
  suggested_user="$(gcloud config get-value account)"
  # TODO(b/109768593): Fix this hack.
  current_context="$(kubectl config current-context)"
  cat <<EOT >> "${TEST_DIR}/install-config.yaml"
contexts:
  - ${current_context}
EOT

  echo "  Uninstalling..."
  /opt/installer/installer \
    --config="${TEST_DIR}/install-config.yaml" \
    --log_dir=/tmp \
    --suggested_user="${suggested_user}" \
    --use_current_context=true \
		--uninstall=deletedeletedelete \
    --vmodule=main=10,configgen=10,kubectl=10,installer=10,exec=10
  echo "  killing kubectl port forward..."
  pkill -f "kubectl -n=nomos-system port-forward.*2222:22" || true
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

  echo "****************** Starting tests ******************"
  local bats_tests=$(
    echo ${TEST_DIR}/e2e.bats;
    find ${TEST_DIR}/testcases -name '*.bats'
  )

  local testcases=()
  if [[ -n ${file_filter} ]]; then
    for file in ${bats_tests}; do
      if echo "${file}" | grep "${file_filter}" &> /dev/null; then
        echo "Will run ${file}"
        testcases+=("${file}")
      fi
    done
  else
    for file in ${bats_tests}; do
      testcases+=("${file}")
    done
  fi

  local retcode=0
  if ! E2E_TEST_FILTER="${testcase_filter}" ${TEST_DIR}/bats/bin/bats ${testcases[@]}; then
    retcode=1
  fi

  end_time=$(date +%s)
  echo "Tests took $(( ${end_time} - ${start_time} )) seconds."
  return ${retcode}
}

filter=""
clean=false
setup=false
while [[ $# -gt 0 ]]; do
  arg=${1}
  case ${arg} in
    --clean)
      clean=true
      shift
    ;;
    --setup)
      setup=true
      shift
    ;;
    --filter)
      filter="${2}"
      shift
      shift
    ;;
    *)
    shift
    ;;
  esac
done

set_up_kube
if $clean ; then
  clean_up
fi
if $setup ; then
  set_up_env
else
  set_up_env_minimal
fi
exitcode=0
if ! main ${filter}; then
  exitcode=1
fi
if $clean ; then
  clean_up
fi
exit ${exitcode}
