#!/bin/bash

# The goal of this script is to choose the correct location for a nomos-manifest.yaml file
#
#
# This task is more complicated than one might expect, as the path to the
# file intends to contain important identifying information about the git
# commit and state of the repository that made up this artifact.

set -euo pipefail

BASE_PATH="gs://config-management-release/nomos-manifest/commits"

# The commit hash is the foundational unit of `git`.  It is bedrock.
CURRENT_COMMIT_HASH=$(git rev-parse HEAD)

# If the repo is dirty, the commit hash does not describe the state of the repository.
# Differentiating this is essential to CURRENT_COMMIT_HASH serving as a strong primitive
# for other build and release tooling.
is_dirty_repo() {
  [ -n "$(git status --short)" ]
  return "$?"
}

COMMIT_IDENTIFIER="$CURRENT_COMMIT_HASH"
if is_dirty_repo; then
  # If the repo is dirty, we want to catalogue this artifact under "commits/dirty/".
  # This keeps "commit/" clean.
  COMMIT_IDENTIFIER="DIRTY/${COMMIT_IDENTIFIER}"

  # We'll also want to label the file with a timestamp, as to differentiate
  # between different uncomitted builds
  TIMESTAMP=$(date --utc +"%Y%m%dT%H%M%S")
  COMMIT_IDENTIFIER="${COMMIT_IDENTIFIER}/${TIMESTAMP}"
fi

echo "${BASE_PATH}/${COMMIT_IDENTIFIER}.yaml"
