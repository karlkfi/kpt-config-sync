#!/bin/bash
#
# This script moves pkg/api/configmanagement to pkg/api/nomos and fixes up code
# generation.  There are still open issues in b/115638903 that need to be
# resolved in order to cleanly move things.
#

set -euo pipefail

set -x

function move() {
  dst="${2:-}"
  mkdir -p "$(dirname "$dst")"
  base=github.com/google/nomos
  gomvpkg -from "$base/${1:-}" -to "$base/${2:-}" \
    -vcs_mv_cmd "git mv {{.Src}} {{.Dst}}"
}

# Move API package
move pkg/api/configmanagement pkg/api/nomos
# Adjust references to client gen packages
move clientgen/apis/typed/configmanagement/v1 clientgen/apis/typed/nomos/v1
move clientgen/informer/configmanagement clientgen/informer/nomos
move clientgen/listers/configmanagement/v1 clientgen/listers/nomos/v1

rm -rf clientgen

sed -i -e 's|configmanagement/v1|nomos/v1|' ./scripts/generate-clientset.sh
./scripts/generate-clientset.sh
./scripts/generate-watcher.sh

go build ./...
go test ./...
