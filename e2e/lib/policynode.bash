#!/bin/bash

# Check if a PolicyNode's syncToken has a specific value.
#
# Arguments:
#   namespace: the namespace name
#   hash: the git hash to compare against
function policynode::sync_token_eq() {
  local namespace="${1:-}"
  local expect="${2:-}"
  local stoken
  stoken="$(kubectl get policynode "${namespace}" -ojsonpath='{.status.syncToken}')"
  if [[ "$stoken" != "$expect" ]]; then
    echo "syncToken is $stoken waiting for $expect"
    return 1
  fi
}
