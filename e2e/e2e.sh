#!/bin/bash
#
# e2e test launcher.  Do not run directly, this is intended to be executed by
# the Makefile for the time being.
#

set -euo pipefail

# If set, the runner will refrain from importing any files from outside of the
# container that uses it.
hermetic=false

EXTRA_ARGS=()
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
    --hermetic)
      hermetic=true
    ;;

    --)
      break
    ;;

    *)
      echo "e2e.sh: unrecognized arg $arg"
      exit 1
    ;;
  esac
done

if "${hermetic}"; then
  echo "+++ Executing e2e tests in hermetic mode."

  # Place the user's home in a writable directory.
  EXTRA_ARGS+=(-e "HOME=/tmp/home/user")
else
  # Copy the gcloud and kubectl configuration into a separate directory then
  # update the gcloud auth provider path to point to the gcloud in the e2e image
  # and add mount points for gcloud / kubectl that will be sandboxed to the
  # container.  Use rsync for copying since we want to ensure the file mods are
  # preserved as to not expose auth tokens.
  #
  # A nice side effect of this is that you can switch context on gcloud / kubectl
  # while tests are running without issues.
  #
  echo "+++ Executing e2e tests in non-hermetic mode."
  rm -rf "$TEMP_OUTPUT_DIR/config"
  mkdir -p "$TEMP_OUTPUT_DIR/config"
  rsync -a ~/.kube "$TEMP_OUTPUT_DIR/config"
  rsync -a ~/.config/gcloud "$TEMP_OUTPUT_DIR/config"
  EXTRA_ARGS+=(-v "$TEMP_OUTPUT_DIR/config/.kube:${HOME}/.kube")
  EXTRA_ARGS+=(-v "$TEMP_OUTPUT_DIR/config/gcloud:${HOME}/.config/gcloud")
  sed -i -e \
    's|cmd-path:.*gcloud$|cmd-path: /opt/gcloud/google-cloud-sdk/bin/gcloud|' \
    "$TEMP_OUTPUT_DIR/config/.kube/config"
  EXTRA_ARGS+=(-e "HOME=${HOME}")
  EXTRA_ARGS+=(-v "${HOME}:${HOME}")
fi


DOCKER_CMD=(
  docker run -it
    --group-add docker # Used for gcloud to get instance metadata.
    -u "$(id -u):$(id -g)"
    -v /var/run/docker.sock:/var/run/docker.sock
    -v "${TEMP_OUTPUT_DIR}:/tmp"
    -v "${OUTPUT_DIR}/e2e":/opt/testing/e2e
    -e "NOMOS_REPO=$(pwd)"
    "${EXTRA_ARGS[@]}"
    "gcr.io/stolos-dev/e2e-tests:test-e2e-latest"
    "$@"
)

"${DOCKER_CMD[@]}"
