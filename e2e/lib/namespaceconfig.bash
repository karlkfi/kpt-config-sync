#!/bin/bash

# Check if a NamespaceConfig's sync token has a specific value.
#
# Arguments:
#   namespace: the namespace name
#   hash: the git hash to compare against
function namespaceconfig::sync_token_eq() {
  local namespace="${1:-}"
  local expect="${2:-}"
  local stoken
  stoken="$(kubectl get namespaceconfig "${namespace}" -ojsonpath='{.status.token}')"
  if [[ "$stoken" != "$expect" ]]; then
    echo "sync token is $stoken waiting for $expect"
    return 1
  fi
}

# Check if a NamespaceConfig has .status.syncState=synced
#
# Arguments:
#   namespace: the namespace name
function namespaceconfig::synced() {
  local namespace="${1:-}"
  status="$(kubectl get namespaceconfig "${namespace}" -ojsonpath='{.status.syncState}')"
  if [[ "$status" != "synced" ]]; then
    echo "sync status is \"$status\" waiting for \"synced\""
    return 1
  fi
}
