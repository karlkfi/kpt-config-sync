#!/bin/bash
#
# e2e test launcher.  Do not run directly, this is intended to be executed by
# the Makefile for the time being.
#

set -euo pipefail

EXTRA_ARGS=()
GCP_PROJECT=stolos-dev
IMAGE_TAG=e2e-latest

while (( $# > 0 )); do
  arg=${1}
  shift
  case ${arg} in
    --TEMP_OUTPUT_DIR)
      TEMP_OUTPUT_DIR="${1:-}"
      shift
    ;;
    --OUTPUT_DIR)
      OUTPUT_DIR="${1:-}"
      shift
    ;;
    --GCP_PROJECT)
      GCP_PROJECT="${1:-}"
      shift
    ;;
    --IMAGE_TAG)
      IMAGE_TAG="${1:-}"
      shift
    ;;
    --IMPORTER)
      IMPORTER="${1:-}"
      shift
    ;;
    --GCP_E2E_WATCHER_CRED)
      GCP_E2E_WATCHER_CRED="${1:-}"
      shift
    ;;
    --GCP_E2E_RUNNER_CRED)
      GCP_E2E_RUNNER_CRED="${1:-}"
      shift
    ;;

    --)
      break
    ;;

    *)
      echo "Unrecognized arg $arg"
      exit 1
    ;;
  esac
done

GCLOUD_PATH="$(dirname "$(command -v gcloud)")"
KUBECTL_PATH="$(dirname "$(command -v kubectl)")"

# Hack around b/111407514 for goobuntu installed with standard
# gcloud/kubectl packages.
if [[ "$KUBECTL_PATH" == "/usr/bin/" ]]; then
  EXTRA_ARGS+=(-v "/usr/bin/kubectl:/usr/bin/kubectl")
fi
EXTRA_ARGS+=(-e "KUBECTL_PATH=${KUBECTL_PATH}")
if [[ "$GCLOUD_PATH" == "/usr/bin" ]]; then
  EXTRA_ARGS+=(-v "/usr/lib/google-cloud-sdk:/usr/lib/google-cloud-sdk")
  EXTRA_ARGS+=(-e "GCLOUD_PATH=/usr/lib/google-cloud-sdk/bin")
else
  EXTRA_ARGS+=(-e "GCLOUD_PATH=${GCLOUD_PATH}")
fi

DOCKER_ARGS=(
  docker run -it
    --group-add docker
    -u "$(id -u):$(id -g)"
    -v /var/run/docker.sock:/var/run/docker.sock
    -v "${TEMP_OUTPUT_DIR}:/tmp"
    -v "${HOME}:${HOME}"
    -v "${OUTPUT_DIR}/e2e":/opt/testing/e2e
    -e "CONTAINER=gcr.io/${GCP_PROJECT}/installer"
    -e "VERSION=${IMAGE_TAG}"
    -e "HOME=${HOME}"
    -e "NOMOS_REPO=$(pwd)"
    "${EXTRA_ARGS[@]}"
    "gcr.io/stolos-dev/e2e-tests:test-e2e-latest"
    --importer "${IMPORTER}"
    --gcp-watcher-cred "${GCP_E2E_WATCHER_CRED}"
    --gcp-runner-cred "${GCP_E2E_RUNNER_CRED}"
    "$@"
)

"${DOCKER_ARGS[@]}"
