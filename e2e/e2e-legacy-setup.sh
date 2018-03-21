#!/bin/bash

# End to end tests for Nomos.
# As a prerequisite for running this test, you must have
# - A 1.9 kubernetes cluster configured for nomos. See
# -- scripts/cluster/gce/kube-up.sh
# -- scripts/cluster/gce/configure-apserver-for-nomos.sh
# -- scripts/cluster/gce/configure-monitoring.sh
# - kubectl configured with context pointing to that cluster
# - Docker
# - gcloud with access to a project that has GCR

# To execute the test without one time setup
# > SKIP_INITIAL_SETUP=1 e2e/e2e.sh

# To execute a subset of tests without setup, run as folows:
# > SKIP_INITIAL_SETUP=1 TEST_FUNCTIONS=testNomosResourceQuota e2e/e2e.sh

# To execute the tests with final clean up
# > SKIP_FINAL_CLEANUP=0 e2e/e2e.sh

set -u

# Should be $STOLOS/e2e
TESTDIR="$( cd "$(dirname "$0")" ; pwd -P )"
# Should be $STOLOS/
MAKEDIR=$TESTDIR/..
# Path to demo acme yaml
ACME=$MAKEDIR/examples/acme/policynodes/acme.yaml

# Run once
function initialSetUp() {
  echo "****************** Setting up environment ******************"
  cd ${MAKEDIR}

  make deploy-common-objects

  if ! kubectl get ns nomos-system > /dev/null; then
    echo "Failed setting up Nomos on cluster"
    exit 1
  fi

  if ! make all-deploy ; then
    echo "Failed to deploy Nomos components, aborting test"
    exit 1
  fi

  kubectl apply -f ${ACME}
  sleep 3
}

initialSetUp

