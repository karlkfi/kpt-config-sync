#!/bin/bash

# Helper script to get our docker tag if we're checked out on a branch
# e.g. 1.4.0.  If we're on master, or if this is a local branch, and we use tag
# "latest".
#
# This lets us stop overwriting gcr.io/config-management-release/nomos:latest
# with images built on branches.
#
# Additional backstory in b/159654901
#
# Get the upstream tracking branch.  Note that this needs to work in multiple
# checkout/fetch environments.  See b/159654901 for the various changes that we
# tried before settling on this.
BRANCH=$(git symbolic-ref --short --quiet HEAD);
if [[ "${BRANCH}" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "${BRANCH}-latest"
else
  echo "latest"
fi
