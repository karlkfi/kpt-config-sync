#!/bin/bash
#
# e2e test launcher.  Do not run directly, this is intended to be executed by
# the Makefile for the time being.
#

set -euo pipefail

# If set, the runner will refrain from importing any files from outside of the
# container that uses it.
hermetic=false

# If set, we will copy the prober creds into the container from gcs.  This is
# useful in hermetic tests where one can not rely on the credentials being
# mounted into the container.
gcs_prober_cred=""

# If set, we will mount the prober creds path into the test runner from here.
mounted_prober_cred=""

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
    --gcs-prober-cred)
      gcs_prober_cred="${1:-}"
      shift
    ;;
    --mounted-prober-cred)
      mounted_prober_cred="${1:-}"
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

echo "+++ Environment: "
env

DOCKER_FLAGS=()
if [[ "${mounted_prober_cred}" != "" ]]; then
  echo "+++ Mounting creds from: ${mounted_prober_cred}"
  DOCKER_FLAGS+=(-v "${mounted_prober_cred}:${mounted_prober_cred}")
fi

# Old-style podutils for go/prow use ${WORKSPACE} as base directory but do not
# define $ARTIFACTS.
if [[ -n "${WORKSPACE+x}" && ! -n "${ARTIFACTS+x}" ]]; then
  ARTIFACTS="${WORKSPACE}/_artifacts"
fi

# The ARTIFACTS env variable has the name of the directory that the test artifacts
# should be written to.  If one is defined, propagate it into the tester.
if [ -n "${ARTIFACTS+x}" ]; then
  echo "+++ Artifacts directory: ${ARTIFACTS}"
  DOCKER_FLAGS+=(
    -e "ARTIFACTS=${ARTIFACTS}"
    -v "${ARTIFACTS}:${ARTIFACTS}"
  )
fi

if "${hermetic}"; then
  # In hermetic mode, the e2e tests do most of the setup required to connect to
  # the test environment.
  #
  # gcloud and kubectl are configured based on the credentials provided by the
  # test runner.   The credentials are either mounted into the container (if
  # the test runner is based on Kubernetes), in which case they are expected in
  # ${TEMP_OUTPUT_DIR}/config/... (see above), or downloaded from GCS using the
  # credentials of the user that is running this wrapper using the flag
  # --gcs-prober-cred if the test runner is based off of local file content.
  echo "+++ Executing e2e tests in hermetic mode."

  rm -rf "${TEMP_OUTPUT_DIR}/user"
  mkdir -p "${TEMP_OUTPUT_DIR}/user"

  # Place the user's home in a writable directory.
  EXTRA_ARGS+=(-e "HOME=/tmp/user")

  # Make the currently checked out directory available.
  EXTRA_ARGS+=(-e "NOMOS_REPO=/tmp/nomos")
  EXTRA_ARGS+=(-v "$(pwd):/tmp/nomos")

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
  EXTRA_ARGS+=(-v "${HOME}:/home/user")
  EXTRA_ARGS+=(-e "NOMOS_REPO=$(pwd)")
fi

if [[ "${gcs_prober_cred}" != "" ]]; then
  echo "+++ Downloading GCS credentials: ${gcs_prober_cred}"
  gsutil cp "${gcs_prober_cred}" "${TEMP_OUTPUT_DIR}/config/prober_runner_client_key.json"
fi


DOCKER_FLAGS+=("--interactive")
if [ -t 0 ]; then
  echo "+++ Docker will use a terminal."
  DOCKER_FLAGS+=("--tty")
fi

DOCKER_FLAGS+=(
    -u "$(id -u):$(id -g)"
    -v "${TEMP_OUTPUT_DIR}:/tmp"
    -v "${OUTPUT_DIR}/e2e":/opt/testing/e2e
    -v "${OUTPUT_DIR}/go/bin":/opt/testing/go/bin
    -e "NOMOS_REPO=$(pwd)"
    "${EXTRA_ARGS[@]}"
    "gcr.io/stolos-dev/e2e-tests:test-e2e-latest"
)

docker run "${DOCKER_FLAGS[@]}" "$@"
