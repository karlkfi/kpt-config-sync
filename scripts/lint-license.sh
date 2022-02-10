#!/bin/bash
# Copyright 2022 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


set -euo pipefail

echo "Checking LICENSES.txt file for changes."
tmp=$(mktemp -d)
filename="LICENSES.txt"

bash scripts/prepare-licenses.sh "${tmp}/${filename}"

if ! cmp -s "${tmp}/${filename}" "${filename}" ; then
  diff "${filename}" "${tmp}/${filename}"
  echo "${filename} file needs updating. To accept changes, run
cp ${tmp}/${filename} ${filename}"
  exit 1
fi
rm -r "${tmp}"

# TODO(b/156962677): Don't directly install things in scripts - do it in the image.
# Remove the commit hash after https://github.com/google/go-licenses/issues/75 is fixed.
go get github.com/google/go-licenses@8751804a5b801cba3064b9823a6e5b4767e1ecdc
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
