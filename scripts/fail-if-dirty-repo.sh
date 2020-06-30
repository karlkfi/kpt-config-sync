#!/bin/bash

set -eou pipefail

is_dirty_repo() {
  [ -n "$(git status --short)" ]
  return "$?"
}

if is_dirty_repo; then
  echo "Cannot push to commit/ from dirty repo"
  echo "Git status output:"
  # Prefix git status output with a pound sign for better readability
  git status | sed 's/.*/# &/'
  exit 1
fi
