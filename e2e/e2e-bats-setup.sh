#!/bin/bash

# End to end tests for Nomos.
# As a prerequisite for running this test, you must have
# - A 1.9 kubernetes cluster configured for nomos. See
# -- scripts/cluster/gce/kube-up.sh
# -- scripts/cluster/gce/configure-apserver-for-nomos.sh
# -- scripts/cluster/gce/configure-monitoring.sh
# - kubectl configured with context pointing to that cluster
# - Docker
# - gcloud with access to a project that has GCR

# To execute a subset of tests without setup, run as folows:
# > SKIP_INITIAL_SETUP=1 TEST_FUNCTIONS=testNomosResourceQuota e2e/e2e.sh

set -u

TEST_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"


######################## MAIN #########################
function kubeSetUp() {
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

function setUpEnv() {
  echo "****************** Setting up environment ******************"
  suggested_user="$(gcloud config get-value account)"

  /opt/testing/init-git-server.sh

  ./installer \
    --config="${TEST_DIR}/install-config.yaml" \
    --log_dir=/tmp \
    --suggested_user="${suggested_user}" \
    --use_current_context=true \
    --vmodule=main=10,configgen=10,kubectl=10,installer=10,exec=10
  echo "****************** Environment is ready ******************"
}

function main() {
  if [[ ! "kubectl get ns > /dev/null" ]]; then
    echo "Kubectl/Cluster misconfigured"
    exit 1
  fi
  GIT_SSH_COMMAND="ssh -q -o StrictHostKeyChecking=no -i /opt/testing/id_rsa.nomos"; export GIT_SSH_COMMAND

  echo "****************** Starting tests ******************"
  ${TEST_DIR}/bats/bin/bats ${TEST_DIR}/e2e.bats
}

function cleanUp() {
  echo "****************** Cleaning up environment ******************"
  kubectl delete ValidatingWebhookConfiguration policy-nodes.nomos.dev --ignore-not-found
  kubectl delete ValidatingWebhookConfiguration resource-quota.nomos.dev --ignore-not-found
  kubectl delete policynodes --all || true
  kubectl delete clusterpolicy --all || true
  kubectl delete --ignore-not-found ns nomos-system
  ! pkill -f "kubectl -n=nomos-system port-forward.*2222:22"

  echo "Deleting namespaces nomos-system, this may take a minute"
  while kubectl get ns nomos-system > /dev/null 2>&1
  do
    sleep 3
    echo -n "."
  done
}

kubeSetUp
cleanUp
setUpEnv
main
if [ "$2" != "noclean" ]
  then
    cleanUp
fi
