#!/bin/bash
#
# Script to clone the nomos operator repo.
#

set -euo pipefail

mkdir -p "$GOPATH/src/github.com/google"
cd "$GOPATH/src/github.com/google"
git clone sso://team/nomos-team/nomos-operator
cd nomos-operator
f="$(git rev-parse --git-dir)/hooks/commit-msg"
curl -Lo "$f" https://gerrit-review.googlesource.com/tools/hooks/commit-msg
chmod +x "$f"
