#!/bin/bash

set -euo pipefail

readonly exclude=(
  scripts/build.sh
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
  scripts/lib/installer.sh
  scripts/nomosvet.sh
  scripts/test-unit.sh
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
