#!/bin/bash
#
# Legacy Testcase Cleanup Scripting

set -euo pipefail

function cleanUp() {
  echo "****************** Cleaning up environment ******************"
  kubectl delete policynodes --all
  kubectl delete ValidatingWebhookConfiguration stolos-resource-quota --ignore-not-found
  kubectl delete --ignore-not-found ns stolos-system

  echo "Deleting namespace stolos-system, this may take a minute"
  while kubectl get ns stolos-system > /dev/null 2>&1
  do
    sleep 3
    echo -n "."
  done
  echo
}

cleanUp
