#!/bin/bash

# Helpers for ignoring command failures.

# ignore::log_err <command>
#
# Executes the given command, capturing any error output but not failing on
# error. Error output will be logged in the event of error.
#
ignore::log_err() {
  # shellcheck disable=SC2155
  local out="$("$@" 2>&1)"
  # shellcheck disable=SC2181
  if [ $? -ne 0 ]; then
    debug::log "$@"
    debug::log "resulted in error:"
    debug::log "${out}"
  fi
  return 0
}