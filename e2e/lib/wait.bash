#!/bin/bash

# Helpers for waiting for async events (e.g. Pods to come up).

# Wait for an event to occur
# Flags
#  -n [namespace] - namespace to fetch event from
#  -r [reason] - event reason
#  -t [timeout] - timeout
#  -d [deadline] - deadline in seconds from epoch
# Args
#  [reason]
function wait::event() {
  local deadline=""
  local namespace=""
  local timeout="10"
  local args=()
  while [[ $# -gt 0 ]]; do
    local arg="${1:-}"
    shift
    case $arg in
      -n)
        namespace=${1:-}
        shift
        ;;
      -t)
        timeout=${1:-}
        shift
        ;;
      -d)
        timeout=""
        deadline=${1:-}
        shift
        ;;
      *)
        args+=("$arg")
        ;;
    esac
  done

  local reason="${args[0]}"
  [ -z "$reason" ] && echo "specify [reason] arg" && false

  if [[ "$timeout" != "" ]]; then
    deadline=$(( $(date +%s) + ${timeout} ))
  fi

  cmd=(
    kubectl get event
  )
  if [[ "$namespace" != "" ]]; then
    cmd+=("-n" "$namespace")
  fi

  while [[ "$(date +%s)" < "${deadline}" ]]; do
    if "${cmd[@]}" | grep "${reason}" &> /dev/null; then
      return 0
    fi
    sleep 0.5
  done
  echo "Failed to get event"
  return 1
}


function wait::for_success() {
  local command="${1:-}"
  local timeout=${2:-10}
  local or_die=${3:-true}
  wait::for "${command}" "${timeout}" "${or_die}" true
}

function wait::for_failure() {
  local command="${1:-}"
  local timeout=${2:-10}
  local or_die=${3:-true}
  wait::for "${command}" "${timeout}" "${or_die}" false
}

# Waits for the command to return with the given status within a timeout period.
#
# Usage:
# wait::for COMMAND TIMEOUT_SEC EXIT_ON_TIMEOUT EXPECTED_STATUS
#
# Example:
#
# `wait::for "sleep 10" 11 true true`
#
function wait::for() {
  local command="${1:-}"
  local timeout=${2:-10}
  local or_die=${3:-true}
  local expect=${4:-true}

  echo -n "Waiting for ${command} to exit ${expect}"
  for i in $(seq 1 ${timeout}); do
    run ${command} &> /dev/null
    if [ $status -eq 0 ]; then
      if ${expect}; then
        echo
        return 0
      fi
    else
      if ! ${expect}; then
        echo
        return 0
      fi
    fi
    echo -n "."
    sleep 0.5
  done
  echo
  echo "Command '${command}' failed after ${timeout} seconds"
  if ${or_die}; then
    exit 1
  fi
}

