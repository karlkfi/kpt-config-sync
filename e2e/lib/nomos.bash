#!/bin/bash

set -euo pipefail

DIR=$(dirname "${BASH_SOURCE[0]}")
# shellcheck source=e2e/lib/debug.bash
source "$DIR/debug.bash"
# shellcheck source=e2e/lib/git.bash
source "$DIR/git.bash"
# shellcheck source=e2e/lib/wait.bash
source "$DIR/wait.bash"
# shellcheck source=e2e/lib/resource.bash
source "$DIR/resource.bash"

# Returns success if the namespace has synced.
# Synced is defined as
#   token == .spec.token && token == .status.token
# No errors is defined as
#   len(.status.errors) == 0
#
# Flags:
#  --error Exit 0 if the namespace is synced and has an error,
#    defined as len(syncErrors) != 0
#
# Arguments:
#  name: the namespace name
function nomos::ns_synced() {
  local args=()
  local error=false
  while (( 0 < $# )); do
    local arg="${1:-}"
    shift
    case "$arg" in
      --error)
        error=true
        ;;
      *)
        args+=("$arg")
        ;;
    esac
  done

  if (( ${#args[@]} != 1 )); then
    debug::error "Invalid number of args for ns_synced"
    return 1
  fi


  local ns="${args[0]}"
  local token
  token="$(git::hash)"
  local output
  local status=0
  output="$(kubectl get namespaceconfig "$ns" -ojson 2>&1)" || status=$?
  if (( status != 0 )); then
    return 1
  fi
  if ! nomos::_synced "$output" "$token" "$error"; then
    return 1
  fi
}

# Returns success if the cluster has synced and there are no errors.
# Synced is defined as
#   token == .spec.token && token == .status.token
# No erros is defined as
#   len(.status.errors) == 0
#
# Flags:
#  --error Exit 0 if the cluster is synced and has an error,
#    defined as len(syncErrors) != 0
#
function nomos::cluster_synced() {
  local args=()
  local error=false
  while (( 0 < $# )); do
    local arg="${1:-}"
    shift
    case "$arg" in
      --error)
        error=true
        ;;
      *)
        args+=("$arg")
        ;;
    esac
  done

  if (( ${#args[@]} != 0 )); then
    echo "Invalid number of args"
    return 1
  fi


  local token
  token="$(git::hash)"
  local output
  local status=0
  output="$(kubectl get clusterconfig config-management-cluster-config -ojson)" || status=$?
  if (( status != 0 )); then
    return 1
  fi
  nomos::_synced "$output" "$token" "$error"
}

# Helper funciton for NS and Cluster synced check.
function nomos::_synced() {
  if (( $# != 3 )); then
    debug::error "Invalid number of args"
    return 1
  fi

  local output="${1:-}"
  local token="${2:-}"
  local error="${3:-}"
  if [[ "$output" == "" ]]; then
    return 1
  fi

  # extract tokens, error count from json output
  local spec_token
  local status_token
  local error_count
  spec_token="$(jq -r '.spec.token' <<< "$output")"
  status_token="$(jq -r '.status.token' <<< "$output")"
  error_count="$(jq '.status.syncErrors | length' <<< "$output")"

  # compare tokens, check error count
  if [[ "$token" != "$spec_token" ]]; then
    return 1
  fi
  if [[ "$token" != "$status_token" ]]; then
    return 1
  fi
  if $error; then
    if ((0 == error_count)); then
      return 1
    fi
  else
    if ((0 < error_count)); then
      return 1
    fi
  fi
}

# Returns success if the repo is fully synced.
# This is defined as
#   token == status.source.token && len(status.source.errors) == 0 \
#   && token == status.import.token && len(status.import.errors) == 0 \
#   && token == status.sync.token && len(status.sync.errors) == 0
#
# Flags:
#  --sourceError    exit 0 if
#     token == status.source.token && 0 < len(status.source.errors)
#  --importError    exit 0 if
#     token == status.source.token && len(status.source.errors) == 0 \
#     && token == status.import.token && 0 < len(status.import.errors)
#  --syncError      exit 0 if
#     token == status.source.token && len(status.source.errors) == 0 \
#     && token == status.import.token && len(status.import.errors) == 0 \
#     && token == status.sync.token && 0 < len(status.sync.errors)
function nomos::repo_synced() {
  local args=()
  local sourceError=false
  local importError=false
  local syncError=false
  local tokenOverride=""

  while (( "$#" )); do
    local arg="${1:-}"
    shift
    case "$arg" in
      --sourceError)
        sourceError=true
        ;;
      --importError)
        importError=true
        ;;
      --syncError)
        syncError=true
        ;;
      --tokenOverride)
        tokenOverride=${1}
        shift
        ;;
      *)
        args+=("$arg")
        ;;
    esac
  done

  if (( ${#args[@]} != 0 )); then
    echo "Invalid number of args for repo_synced"
    return 1
  fi

  local token
  if [ -n "${tokenOverride}" ]; then
    token="${tokenOverride}"
  else
    token="$(git::hash)"
  fi

  local output
  # This fixes a race condition for when the repo object is deleted from git SOT
  # as part of the testing scenario
  wait::for -t 60 -p 1 -- resource::check repo repo
  output="$(kubectl get repo repo -ojson)"

  local status_token
  local error_count
  status_token="$(jq -r '.status.source.token' <<< "$output")"
  error_count="$(jq '.status.source.errors | length' <<< "$output")"
  if [[ "$token" != "$status_token" ]]; then
    return 1
  fi
  if $sourceError; then
    if (( 0 == error_count )); then
      return 1
    fi
    return 0
  fi
  if (( 0 < error_count )); then
    return 1
  fi

  status_token="$(jq -r '.status.import.token' <<< "$output")"
  error_count="$(jq '.status.import.errors | length' <<< "$output")"
  if [[ "$token" != "$status_token" ]]; then
    return 1
  fi
  if $importError; then
    if (( 0 == error_count )); then
      return 1
    fi
    return 0
  fi
  if (( 0 < error_count )); then
    return 1
  fi

  status_token="$(jq -r '.status.sync.latestToken' <<< "$output")"
  error_count="$(jq '[.status.sync.inProgress[].errors] | flatten | length' <<< "$output")"
  if [[ "$token" != "$status_token" ]]; then
    return 1
  fi
  if $syncError; then
    if (( 0 == error_count )); then
      return 1
    fi
  fi

  #### WORKAROUND ####
  # Right now we do not yet report status correctly on deleting namespaces.
  # This check will ensure that we do not report synced if there are still
  # managed namespaces which do not have an associated NamespaceConfig object.
  ####################
  local ns_count
  local nc_count
  ns_count="$(
    kubectl get ns -ojson \
    | jq '[.items[] | select(.metadata.annotations["configmanagement.gke.io/managed"] == "enabled") ] | length'
  )"
  nc_count="$(kubectl get nc -ojson | jq '.items | length')"
  if (( ns_count != nc_count )); then
    return 1
  fi
  #### END WORKAROUND ####

  if (( 0 < error_count )); then
    return 1
  fi
}

# Restarts the git-importer, monitor, and syncer pods
# Without the operator, it's necessary to restart these pods for them to pickup changes
# to their configMaps
function nomos::restart_pods() {
  kubectl delete pod -n config-management-system -l app=git-importer
  kubectl delete pod -n config-management-system -l app=monitor
  kubectl delete pod -n config-management-system -l app=syncer
}
