#!/bin/bash

#
# golang e2e test launcher. Do not run directly, this is intended to be executed by the prow job inside a container.

set -eo pipefail

DIR=$(dirname "${BASH_SOURCE[0]}")

if [ $KUBERNETES_ENV == "GKE" ]; then
  echo "GKE enviornment"

  # Makes the service account from ${gcp_prober_cred} the active account that drives
  # cluster changes.
  gcloud --quiet auth activate-service-account --key-file="${GCP_PROBER_CREDS}"

  # Installs gcloud as an auth helper for kubectl with the credentials that
  # were set with the service account activation above.
  # Needs cloud.containers.get permission.
  # shellcheck disable=SC2154
  gcloud --quiet container clusters get-credentials "${GCP_CLUSTER}" --zone="${GCP_ZONE}" --project="${GCP_PROJECT}"
elif [ $KUBERNETES_ENV == "KIND" ]; then
  echo "kind environment"
else
  echo "Unexpected Kubernetes environment, '$KUBERNETES_ENV'"
  exit 1
fi

set +e

echo "Starting e2e tests"
start_time=$(date +%s)
go test ./e2e/... --e2e --test.v "$@" | tee test_results.txt
exit_code=$?
end_time=$(date +%s)
echo "Tests took $(( end_time - start_time )) seconds"

if [ -d "/logs/artifacts" ]; then
  echo "Creating junit xml report"
  cat test_results.txt | go-junit-report > /logs/artifacts/junit_report.xml
fi

exit $exit_code
