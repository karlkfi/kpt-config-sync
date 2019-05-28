#!/bin/bash

set -euo pipefail

# bats tests aren't really bash, so exclude them.
readonly exclude_bats=(
  e2e/testcases/acme.bats
  e2e/testcases/basic.bats
  e2e/testcases/cluster_resources.bats
  e2e/testcases/namespaces.bats
)

# mapfile reads stdin lines into array, -t trims newlines
mapfile -t files < <(
  find scripts e2e -type f \( -name '*.sh' -o -name '*.bash' \)
)
mapfile -t check_files < <(
  echo "${files[@]}" \
    | tr ' ' '\n' \
    | sort \
    | uniq -u
)

# Handle bats tests
bats_tmp="$(mktemp -d lint-bash-XXXXXX)"
function cleanup() {
  rm -rf "${bats_tmp}"
}
trap cleanup EXIT
mapfile -t bats_tests < <(find e2e scripts -type f -name '*.bats')
mapfile -t check_bats < <(
  echo "${bats_tests[@]}" "${exclude_bats[@]}" \
    | tr ' ' '\n' \
    | sort \
    | uniq -u
)

export BATS_TEST_PATTERN="^[[:blank:]]*@test[[:blank:]]+(.*[^[:blank:]])[[:blank:]]+\\{(.*)\$"
if (( 0 < ${#check_bats[@]} )); then
  for f in "${check_bats[@]}"; do
    dest="${bats_tmp}/$f"
    mkdir -p "$(dirname "$dest")"
    third_party/bats-core/libexec/bats-core/bats-preprocess \
      <<< "$(< "$f")"$'\n' \
      > "${dest}"
    check_files+=("${dest}")
  done
fi

readonly linter=koalaman/shellcheck:v0.6.0

if ! docker image inspect "$linter" &> /dev/null; then
  docker pull "$linter"
fi

cmd=(docker run -v "$(pwd):/mnt")
if [ -t 1 ]; then
  cmd+=(--tty)
fi
cmd+=(
  --rm
  "$linter" "${check_files[@]}"
)


echo "Linting scripts..."
"${cmd[@]}"
echo "PASS"
