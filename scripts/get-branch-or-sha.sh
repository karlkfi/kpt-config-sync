#!/bin/bash

set -eou pipefail

CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD)
COMMIT_SHA=$(git rev-parse HEAD)

BRANCH_OR_SHA=$CURRENT_BRANCH

if [ "$CURRENT_BRANCH" = "HEAD" ]; then
  BRANCH_OR_SHA=$COMMIT_SHA
fi

echo "$BRANCH_OR_SHA"
