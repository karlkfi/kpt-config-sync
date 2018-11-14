#!/bin/bash

# Helpers for waiting for async events (e.g. Pods to come up).

# Waits for the command finish with a certain exit code.
#
# Wait for success:
#  wait:for -- <command>
# Wait for failure:
#  wait::for -f -- <command>
#
# Flags
#   -s                 Wait for success (exit 0)
#   -f                 Wait for failure (exit nonzero)
#   -e [exit code]     Wait for specific integer exit code
#   -t [timeout]       The timeout in seconds (default 30 seconds)
#   -d [deadline]      The deadline in seconds since the epoch
#   -p [poll interval] The amount of time to wait between executions
#   -- End of flags, command starts after this
# Args
#  Args for command
function wait::for() {
  local args=()
  local sleeptime="0.1"
  local exitf=(wait::__exit_eq 0)
  local timeout=300
  local deadline="$(( $(date +%s) + timeout ))"

  local parse_args=false
  for i in "$@"; do
    if [[ "$i" == "--" ]]; then
      parse_args=true
    fi
  done

  if $parse_args; then
    while [[ $# -gt 0 ]]; do
      local arg="${1:-}"
      shift
      case $arg in
        -s)
          exitf=(wait::__exit_eq 0)
        ;;
        -f)
          exitf=(wait::__exit_neq 0)
        ;;
        -e)
          exitf=(wait::__exit_eq "${1:-}")
          shift
        ;;
        -t)
          timeout="${1:-}"
          deadline=$(( $(date +%s) + timeout ))
          shift
        ;;
        -d)
          deadline=${1:-}
          timeout=$(( deadline - $(date +%s) ))
          shift
        ;;
        -p)
          sleeptime=${1:-}
          shift
        ;;
        --)
          break
        ;;
        *)
          echo "Unexpected arg $arg" >&3
          return 1
        ;;
      esac
    done
  fi
  args=("$@")

  local out=""
  local status=0
  while (( $(date +%s) < deadline )); do
    status=0
    out="$("${args[@]}" 2>&1)" || status=$?
    if "${exitf[@]}" "${status}"; then
      return 0
    fi
    sleep "$sleeptime"
  done

  echo "Command timed out after ${timeout} sec:" \
    "${args[@]}" "status: ${status} last output: ${out}"
  return 1
}

function wait::__exit_eq() {
  local lhs="${1:-}"
  local rhs="${2:-}"
  (( lhs == rhs ))
}

function wait::__exit_neq() {
  local lhs="${1:-}"
  local rhs="${2:-}"
  (( lhs != rhs ))
}

