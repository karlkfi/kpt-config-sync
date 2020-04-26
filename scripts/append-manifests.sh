#!/bin/bash
#
# Copyright 2018 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
# Use this script to cat yaml k8s specs together for fun and profit.
# The first argument is a directory containing all the manifests you wish to
# concatenate.
# The second argument is file to output to.
# The third argument is the file pattern from within the directory to use.
# Usage:
# ./append_manifests.sh [INPUT_DIR] [OUTPUT_FILE] [PATTERN]

input_dir=$1
output_file=$2

# Use pattern to limit only to specific files, for example:
# ./append_manifests.sh /some/dir /some/other/dir/file.yaml 'foo*.yaml'
pattern="${3:-}"

# This is a list of files to not copy. It would make more sense in generate.sh,
# but there is not really a good way to export a bash array into the environment
# of a child, so it is left here for now.
# Citation: https://www.mail-archive.com/bug-bash@gnu.org/msg01774.html
declare -A ignoreFiles=(
#  ["deployment/gcp-importer.yaml"]=1
#  ["00-namespace.yaml"]=1
#  ["README.md"]=1
)

# Filters all the files in $input_dir by grepping their names against a pattern.
# Note that it does this against the filename, not the full filepath
for manifest in $(find "${input_dir}" -type f -print0 | xargs -0 basename -a | grep "${pattern}" | sort)
do
  if [[ -z "${ignoreFiles[$manifest]}" ]]
  then
    { echo "# ----- $manifest -----";
      cat "$input_dir/$manifest";
      echo ""; echo "---";
    } >> "$output_file"
  fi
done
