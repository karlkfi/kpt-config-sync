#!/bin/bash

# End to end tests for Stolos.
# As a prerequisite for running this test, you must have
# - A 1.9 kubernetes cluster configured for stolos. See
# -- scripts/cluster/gce/kube-up.sh
# -- scripts/cluster/gce/configure-apserver-for-stolos.sh
# -- scripts/cluster/gce/configure-monitoring.sh
# - kubectl configured with context pointing to that cluster
# - Docker
# - gcloud with access to a project that has GCR

# To execute the test without one time setup
# > SKIP_INITIAL_SETUP=1 e2e/e2e.sh

# To execute a subset of tests without setup, run as folows:
# > SKIP_INITIAL_SETUP=1 TEST_FUNCTIONS=testStolosResourceQuota e2e/e2e.sh

# To execute the tests without final clean up
# > SKIP_FINAL_CLEANUP=1 e2e/e2e.sh

set -euo pipefail

REPO=$(dirname $(readlink -f $0))/..
BIN=stolos-end-to-end

# Build e2e binary
cd ${REPO}
go install ./cmd/${BIN}

# Find the binary in gopath
IFS=':'
EXEC=$(find ${GOPATH} -type f -name $BIN | head -n1)
IFS=''

# Execute e2e binary
${EXEC} -repo_dir ${REPO} "$@"
