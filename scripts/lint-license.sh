#!/bin/bash

set -euxo pipefail

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

echo "Ensuring dependencies contain approved licenses. This may take up to 3 minutes."
binaries=(nomos git-importer syncer monitor)
for binary in ${binaries[*]}
do
  echo "Linting github.com/google/nomos/cmd/${binary}"
  # Recursively checks dependencies of the passed binary.
  # Filter out log messages related to not finding a LICENSE file in literally
  # every nomos directory.
  go run github.com/google/go-licenses check github.com/google/nomos/cmd/"${binary}" \
    2> >(grep -v "Failed to find license for github.com/google/nomos/")
done
