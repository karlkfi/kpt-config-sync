#!/bin/bash
# This script runs the prober e2e test script using the current user's name,
# the test runner credentials, and the prober test cluster.
IMAGE="gcr.io/stolos-dev/e2e-prober:test-e2e-latest"
PROBER_CLIENT_KEY_GS="gs://stolos-dev/e2e/nomos-e2e.joonix.net/prober_runner_client_key.json"
gsutil cp "${PROBER_CLIENT_KEY_GS}" "${HOME}"
PROBER_SERVICE_ACCOUNT_FILE="/etc/prober-gcp-service-account/prober_runner_client_key.json"

docker run -it \
  --env="USER=${USER}" \
  --volume="${HOME}/prober_runner_client_key.json:${PROBER_SERVICE_ACCOUNT_FILE}" \
  "${IMAGE}" "$@"

