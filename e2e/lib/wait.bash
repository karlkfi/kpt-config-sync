#!/bin/bash

# Helpers for waiting for async events (e.g. Pods to come up).

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

