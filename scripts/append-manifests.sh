#!/bin/bash

# Use this script to cat yaml k8s specs together for fun and profit.
# The first argument is a directory containing all the manifests you wish to
# concatenate.
# The second argument is file to output to.
# The third argument is the file pattern from within the directory to use.
# Usage:
# ./append_manifests.sh [INPUT_DIR] [OUTPUT_FILE] [PATTERN]

set -eo pipefail

input_dir=$1
output_file=$2

# Use pattern to limit only to specific files, for example:
# ./append_manifests.sh /some/dir /some/other/dir/file.yaml 'foo*.yaml'
pattern="${3:-}"

# This is a list of files to not copy. It would make more sense in generate.sh,
# but there is not really a good way to export a bash array into the environment
# of a child, so it is left here for now.
# Citation: https://www.mail-archive.com/bug-bash@gnu.org/msg01774.html
# TODO (b/161182417): Make this something we can pass in.  This is hard to use.
declare -A ignoreFiles=(
#  ["deployment/gcp-importer.yaml"]=1
 ["00-namespace.yaml"]=1
#  ["README.md"]=1
)

# Calls to this script may be chained on the same file, so we add a delineator at the beginning
# based on:
#   a) file existing and being not empty
#   b) the last line of that file being a "---"
delineator=""
if [[ -s "$output_file" ]]; then
  last_line=$(tail -n 1 "$output_file")
  if [[ "$last_line" != "---" ]]; then
    delineator="---"
  fi
fi

# Filters all the files in $input_dir by grepping their names against a pattern.
# Note that it does this against the filename, not the full filepath
for manifest in $(find "${input_dir}" -type f -print0 | xargs -0 basename -a | grep "${pattern}" | sort)
do
  if [[ -z "${ignoreFiles[$manifest]}" ]]
  then
    # Only adding the delineator if it is non-empty prevents an empty line at the beginning of the file
    if [ -n "$delineator" ]; then
      echo "$delineator" >> "$output_file"
    fi

    { echo "# ----- $manifest -----";
      cat "$input_dir/$manifest";
    } >> "$output_file"

    delineator="---"
  fi
done
