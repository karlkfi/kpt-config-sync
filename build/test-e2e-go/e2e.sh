#!/bin/bash

#
# golang e2e test launcher. Do not run directly, this is intended to be executed by the prow job inside a container.

set -e

DIR=$(dirname "${BASH_SOURCE[0]}")

# Makes the service account from ${gcp_prober_cred} the active account that drives
# cluster changes.
gcloud --quiet auth activate-service-account --key-file="${GCP_PROBER_CREDS}"

# Installs gcloud as an auth helper for kubectl with the credentials that
# were set with the service account activation above.
# Needs cloud.containers.get permission.
# shellcheck disable=SC2154
gcloud --quiet container clusters get-credentials "${GCP_CLUSTER}" --zone="${GCP_ZONE}" --project="${GCP_PROJECT}"

start_time=$(date +%s)

function test_time {
  end_time=$(date +%s)
  echo "Tests took $(( end_time - start_time )) seconds."
  kill 0 # kill the current process, and all of the processes in the process group
}

trap test_time EXIT

# A hacky way to keep the gcloud credentials up-to-date.
# Run the refresh script in background.
${DIR}/refresh-gcloud-creds.sh &

# Start the e2e test
go test ./e2e/... --e2e "$@"
