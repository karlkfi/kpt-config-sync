#!/bin/bash
#
# This script automates mirroring the "foo-corp" example to our github repo
# under a branch that matches the current repo version for nomos.
#
# Example usage:
# ./scripts/mirror-foo-corp-to-gh.sh
#

set -euo pipefail
set -x

REMOTE=git@github.com:GoogleCloudPlatform/csp-config-management.git

ROOT=$(git rev-parse --show-toplevel)
COMMIT_MESSAGE=""
REPO_VERSION=""
REPO=""

# This remote branch needs to already exist.
# Manually create it if necessary.
while (( 0 < $# )); do
  arg="${1:-}"
  shift
  case $arg in
    -v|--version)
      REPO_VERSION="${1:-}"
      shift
      ;;
    -m|--message)
      COMMIT_MESSAGE="${1:-}"
      shift
      ;;
    -r|--repo)
      REPO="${1:-}"
      shift
      ;;
    *)
      echo "invalid arg $arg"
      exit 1
      ;;
  esac
done
if [[ "$REPO_VERSION" == "" ]]; then
  REPO_VERSION="$(go run "$ROOT/cmd/repo-version")"
fi
if [[ "$COMMIT_MESSAGE" == "" ]]; then
  version=$(git describe --tags --always --dirty)
  COMMIT_MESSAGE="Synced from $version"
fi

function cleanup() {
  rm -rf "$REPO"
}

if [[ "$REPO" == "" ]]; then
  REPO=$(mktemp -d /tmp/foo-corp-example-XXXXX)
  trap cleanup EXIT
  git clone "$REMOTE" "$REPO"
fi

if ! git -C "$REPO" checkout -t "origin/$REPO_VERSION" &> /dev/null; then
  git -C "$REPO" checkout -B "$REPO_VERSION"
fi

EXAMPLE="$(dirname "$0")/../examples/foo-corp-example/"
rsync -rv --delete --exclude '.git' --exclude 'OWNERS.nomos' "$EXAMPLE" "$REPO"
git -C "$REPO" add .
git -C "$REPO" commit -m "$COMMIT_MESSAGE"
git -C "$REPO" push -u origin "HEAD:$REPO_VERSION"
