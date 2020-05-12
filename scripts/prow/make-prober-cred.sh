#!/bin/bash
#
# Our e2e job currently relies on a "prober cred" cert for talking to cloud.
# I'm not sure why it's called prober cred, but it is.  We have to create this
# in our prow cluster in order for e2e to work.
#
# This is the script that was used to set up the
# nomos-prober-runner-gcp-client-key secret in the team prow cluster inside the
# test-pods namespace.
#
# Note: b/156379681 has been filed to clean up this behavior, please check
# status of that bug before using.
#

set -euo pipefail

gsutil cp gs://stolos-dev/e2e/nomos-e2e.joonix.net/prober_runner_client_key.json ./
kubectl create secret generic \
  nomos-prober-runner-gcp-client-key \
  -n test-pods \
  --from-file prober_runner_client_key.json
