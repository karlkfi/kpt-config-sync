#!/bin/bash
#
# e2e test launcher.  Do not run directly, this is intended to be executed by
# the Makefile for the time being.
#

set -euo pipefail

[ -z "${GOTOPT2_BINARY}" ] && (echo "no gotopt2"; exit 1)
readonly gotopt2_output=$(${GOTOPT2_BINARY} "${@}" <<EOF
flags:
- name: "TEMP_OUTPUT_DIR"
  type: string
  help: "The directory for temporary output"
- name: "OUTPUT_DIR"
  type: string
  help: "The directory for temporary output"
- name: "gcs-prober-cred"
  type: string
  help: "If set, we will copy the prober creds into the container from gcs.  This is useful in hermetic tests where one can not rely on the credentials being mounted into the container"
- name: "mounted-prober-cred"
  type: string
  help: "If set, we will mount the prober creds path into the test runner from here"
- name: "hermetic"
  type: bool
  help: "If set, the runner will refrain from importing any files from outside of the container that uses it"
EOF
)
eval "${gotopt2_output}"

# TODO(filmil): remove the need to disable lint checks here.
# shellcheck disable=SC2154
readonly TEMP_OUTPUT_DIR="${gotopt2_TEMP_OUTPUT_DIR}"
# shellcheck disable=SC2154
readonly OUTPUT_DIR="${gotopt2_OUTPUT_DIR}"
# shellcheck disable=SC2154
readonly gcs_prober_cred="${gotopt2_gcs_prober_cred}"
# shellcheck disable=SC2154
readonly mounted_prober_cred="${gotopt2_mounted_prober_cred}"
# shellcheck disable=SC2154
readonly hermetic="${gotopt2_hermetic:-false}"

if [[ "$OUTPUT_DIR" == "" ]]; then
  pushd "$(readlink -f "$(dirname "$0")/..")" > /dev/null
  OUTPUT_DIR="$(make print-OUTPUT_DIR)"
  popd > /dev/null
fi
if [[ "$TEMP_OUTPUT_DIR" == "" ]]; then
  TEMP_OUTPUT_DIR="$OUTPUT_DIR/tmp"
fi

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
  echo "+++ Got legacy artifacts directory from workspace: ${ARTIFACTS}"
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
  DOCKER_FLAGS+=(-e "HOME=/tmp/user")

  # Make the currently checked out directory available.
  DOCKER_FLAGS+=(-e "NOMOS_REPO=/tmp/nomos")
  DOCKER_FLAGS+=(-v "$(pwd):/tmp/nomos")

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
  mkdir -p "$TEMP_OUTPUT_DIR/config/.kube"
  rsync -a ~/.kube "$TEMP_OUTPUT_DIR/config"
  rsync -a ~/.config/gcloud "$TEMP_OUTPUT_DIR/config"
  DOCKER_FLAGS+=(-v "$TEMP_OUTPUT_DIR/config/.kube:${HOME}/.kube")
  DOCKER_FLAGS+=(-v "$TEMP_OUTPUT_DIR/config/gcloud:${HOME}/.config/gcloud")
  sed -i -e \
    's|cmd-path:.*gcloud$|cmd-path: /opt/gcloud/google-cloud-sdk/bin/gcloud|' \
    "$TEMP_OUTPUT_DIR/config/.kube/config"
  DOCKER_FLAGS+=(-e "HOME=${HOME}")
  DOCKER_FLAGS+=(-v "${HOME}:${HOME}")
  DOCKER_FLAGS+=(-e "NOMOS_REPO=$(pwd)")
fi

if [[ "${gcs_prober_cred}" != "" ]]; then
  echo "+++ Downloading GCS credentials: ${gcs_prober_cred}"
  gsutil cp "${gcs_prober_cred}" "${TEMP_OUTPUT_DIR}/config/prober_runner_client_key.json"
fi

DOCKER_FLAGS+=(
    -u "$(id -u):$(id -g)"
    -v "${TEMP_OUTPUT_DIR}:/tmp"
    -v "${OUTPUT_DIR}/e2e":/opt/testing/e2e
    -v "${OUTPUT_DIR}/go/bin":/opt/testing/go/bin
    "gcr.io/stolos-dev/e2e-tests:test-e2e-latest"
)

# shellcheck disable=SC2154
docker run "${DOCKER_FLAGS[@]}" "/opt/testing/e2e/setup.sh" "${gotopt2_args__[@]}"
