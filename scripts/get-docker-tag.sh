#!/bin/sh

# Helper script to get our docker tag if we're checked out on a branch
# e.g. 1.4.0.  If we're on master, and we use tag "latest".
#
# This lets us stop overwriting gcr.io/config-management-release/nomos:latest
# with images built on branches.

BRANCH=$(git rev-parse --abbrev-ref HEAD)
if [ "${BRANCH}" = "master" ]; then
  echo "latest"
else
  echo "${BRANCH}-latest"
fi
