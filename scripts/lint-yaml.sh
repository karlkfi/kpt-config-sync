#!/bin/bash

# KubeBuilder and various YAML writers add invalid fields that we have to
# remember to remove. This linter makes sure we remember to remove the common
# bad fields it adds.

# Use status to keep track of whether we've encountered validation errors.
# We want to show all validation errors instead of exiting early.
status=0

validate() {
  # While these aren't perfect heuristics, they're good enough for all cases we
  # have.
  dir=$1

  if grep -nrP '^status:' "${dir}" --include="*.yaml"; then
    status=1
    echo "Found illegal status field declarations."
    echo "Delete from above files."
  fi

  if grep -nrP '^  creationTimestamp:' "${dir}" --include="*.yaml"; then
    status=1
    echo "Found illegal creationTimestamp field declarations."
    echo "Delete from above files."
  fi
}

validate manifests/
validate e2e/testdata/

exit "${status}"
