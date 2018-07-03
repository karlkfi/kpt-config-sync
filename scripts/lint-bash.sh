#!/bin/bash

set -euo pipefail

if ! command -v shellcheck &> /dev/null; then
  echo "Requires shellcheck, please run:"
  echo "sudo apt-get install shellcheck"
  exit 1
fi

exclude=(
  e2e/lib/assert.bash
  e2e/lib/debug.bash
  e2e/lib/git.bash
  e2e/lib/loader.bash
  e2e/lib/namespace.bash
  e2e/lib/resource.bash
  e2e/lib/setup.bash
  e2e/lib/wait.bash
  e2e/setup.sh
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
  scripts/lint.sh
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

echo "Linting scripts..."
shellcheck "${check_files[@]}"
echo "PASS"
