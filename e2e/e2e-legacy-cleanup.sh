#!/bin/bash
#
# Legacy Testcase Cleanup Scripting

set -euo pipefail

function cleanUp() {
  echo "****************** Cleaning up environment ******************"
  kubectl delete policynodes --all || true
  kubectl delete ValidatingWebhookConfiguration resource-quota.nomos.dev --ignore-not-found
  kubectl delete --ignore-not-found ns nomos-system

  echo "Deleting namespace nomos-system, this may take a minute"
  while kubectl get ns nomos-system > /dev/null 2>&1
  do
    sleep 3
    echo -n "."
  done
  echo
}

cleanUp
