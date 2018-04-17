#!/bin/bash
#
# Legacy Testcase Cleanup Scripting

set -euo pipefail

function cleanUp() {
  echo "****************** Cleaning up environment ******************"
  kubectl delete ValidatingWebhookConfiguration policy-nodes.nomos.dev --ignore-not-found
  kubectl delete ValidatingWebhookConfiguration resource-quota.nomos.dev --ignore-not-found
  kubectl delete policynodes --all || true
  kubectl delete clusterpolicy --all || true
  kubectl delete --ignore-not-found ns nomos-system
  kubectl delete --ignore-not-found ns nomos-system-test
  ! pkill -f "kubectl -n=nomos-system-test port-forward.*2222:22"

  echo "Deleting namespaces nomos-system and nomos-system-test, this may take a minute"
  while kubectl get ns nomos-system > /dev/null 2>&1
  do
    sleep 3
    echo -n "."
  done
  while kubectl get ns nomos-system-test > /dev/null 2>&1
  do
    sleep 3
    echo -n "."
  done
  echo
}

cleanUp
