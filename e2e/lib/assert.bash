#!/bin/bash

# Helpers for test assertions.

# assert::contains <command> <substring>
#
# Will fail if the output of the command or its error message doesn't contain substring
#
function assert::contains() {
  if [[ "$output" != *"$1"* ]]; then
    echo "FAIL: [$output] does not contain [$1]"
    false
  fi
}

