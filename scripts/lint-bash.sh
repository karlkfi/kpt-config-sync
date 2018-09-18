#!/bin/bash

set -euo pipefail

readonly exclude=(
  scripts/cluster/gce/configure-monitoring.sh
  scripts/cluster/gce/download-k8s-release.sh
  scripts/cluster/gce/full-setup.sh
  scripts/cluster/gce/gce-common.sh
  scripts/cluster/gce/gencerts.sh
  scripts/cluster/gce/generate_diff.sh
  scripts/cluster/gce/install.sh
  scripts/cluster/gce/kube-down.sh
  scripts/cluster/gce/kube-up.sh
  scripts/deploy-gcp-importer.sh
  scripts/deploy-policy-admission-controller.sh
  scripts/deploy-resourcequota-admission-controller.sh
  scripts/docs-generate.sh
  scripts/generate-admission-controller-certs.sh
  scripts/generate-clientset.sh
  scripts/generate-policy-admission-controller-certs.sh
  scripts/generate-resourcequota-admission-controller-certs.sh
  scripts/generate-watcher.sh
  scripts/init-git-server.sh
  scripts/test-unit.sh
)
readonly exclude_bats=(
  e2e/testcases/acme.bats
  e2e/testcases/basic.bats
  e2e/testcases/cluster_resources.bats
  e2e/testcases/namespaces.bats
  e2e/gcp_testcases/basic.bats
)

# mapfile reads stdin lines into array, -t trims newlines
mapfile -t files < <(
  find scripts e2e -type f \( -name '*.sh' -o -name '*.bash' \)
)
mapfile -t check_files < <(
  echo "${files[@]}" "${exclude[@]}" \
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
    third_party/bats-core/libexec/bats-preprocess \
      <<< "$(< "$f")"$'\n' \
      > "${dest}"
    check_files+=("${dest}")
  done
fi

readonly linter=koalaman/shellcheck:v0.5.0

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
