#!/bin/bash

set -euo pipefail

echo "Checking LICENSE file for changes."
tmp=$(mktemp -d)

bash scripts/prepare-licenses.sh "${tmp}/LICENSE"

if ! cmp -s "${tmp}/LICENSE" "LICENSE" ; then
  diff "LICENSE" "${tmp}/LICENSE"
  echo "LICENSE file needs updating. To accept changes, run
cp ${tmp}/LICENSE LICENSE"
  exit 1
fi
rm -r "${tmp}"

# TODO(b/156962677): Don't directly install things in scripts - do it in the image.
go get github.com/google/go-licenses
# go mod tidy after getting go-licenses so that it is installed, but not
# declared as a project dependency.
go mod tidy

chmod -R +rwx .output

echo "Ensuring dependencies contain approved licenses. This may take up to 3 minutes."
# Lints licenses for all packages, even ones just for testing.
# Faster than evaluating the dependencies of each individual binary by about 3x.
# We want word-splitting here rather than manually looping through the packages one-by-one.
# shellcheck disable=SC2046
go-licenses check $(go list all)
