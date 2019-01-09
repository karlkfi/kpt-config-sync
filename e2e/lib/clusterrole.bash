#!/bin/bash

# Check if a ClusterRole's rules count is a specific amount.
#
# Arguments:
#   name: the ClusterRole name
#   count: the expected rules count
function clusterrole::rules_count_eq() {
  local name="${1:-}"
  local count="${2:-}"
  local actualcount
  actualcount=$(kubectl get clusterrole "${name}" -ojson | jq -c ".rules | length")
  if (( "$count" != "$actualcount" )); then
    echo "clusterroles.rules count is $actualcount waiting for $count"
    return 1
  fi
}