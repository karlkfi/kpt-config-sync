#!/bin/bash
#
# Copyright 2018 The Stolos Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Releaser.
#
# Invoke as:
#
#     STAGING_DIR=/some/directory \
#     VERSION=1.2.3 \
#     RELEASE_BUCKET=gs://stolos-release/release \
#     STABLE=true \
#         ./release.sh

set -euo pipefail

source "${PWD}/third_party/cloudflare/semver_bash/semver.sh"

# This is the staging directory where all output artifacts are located.
# Required.
STAGING_DIR=${STAGING_DIR:-""}

# This is the semantic version that will be released.  Required.
VERSION=${VERSION:-""}

# This is the GCP release bucked that we will release into.
RELEASE_BUCKET=${RELEASE_BUCKET:-""}

# Set to nonempty value to denote this release as "stable".
STABLE=${STABLE:-}

# Copies the set of artifacts with the glob pattern in $1 to destination
# directory $2.
function release::stage_artifacts() {
  echo "+++ Staging artifacts"

  local srcs="$1"
  local dest="$2"

  mkdir -p "${dest}"
  cp -R ${srcs} "${dest}"
}

# Archives all individual directories in INPUT_DIR into separate archives.
function release::archive_artifacts() {
  echo "+++ Archiving artifacts"
  local input_dir="$1"
  cd "${input_dir}"
  for artifact in $(ls -d *); do
    tar --create --gzip --file "${OUTPUT_DIR}/${artifact}.tar.gz" ${artifact}/*
  done
  cd -
}

# Uploads all artifacts from OUTPUT_DIR to the release bucket on GCS.
function release::upload_artifacts() {
  echo "+++ Uploading artifacts"
  local artifacts=$(find ${OUTPUT_DIR}/*)
  gsutil -m cp ${artifacts} ${RELEASE_BUCKET}/${VERSION} 2>&1 | sed 's/^/---\t/'
}

# Reads the version information from the remote GCS file corresponding to
# version in $1.
function release::get_semver() {
  local remote_file=$1
  # Any error output is interpreted as a missing file.
  gsutil cat "${RELEASE_BUCKET}/${remote_file}" || echo "0.0.0"
}

# Returns a list of version files for semver string in $1.
# Example:
#   release::version_files "1.2.3" --> "latest.txt latest-1.txt latest-1.2.txt \
#                                       stable.txt stable-1.txt stable-1.2.txt"
function release::version_files {
  local version=$1
  local major=0
  local minor=0
  local patch=0
  local special=""
  semverParseInto ${VERSION} major minor patch special

  local files="latest.txt latest-${major}.txt latest-${major}.${minor}.txt"
  if [[ ${STABLE} -ne "" ]]; then
    files+=" stable.txt stable-${major}.txt stable-${major}.${minor}.txt"
  fi
  echo ${files}
}

# Uploads updated version information to the release metadata files.  Only
# files that are affected by the update will be modified.
#
# Release metadata files are:
#   latest.txt
#   latest-1.txt
#   latest-1.2.txt
#   stable.txt
#   stable-1.txt
#   stable-1.2.txt
#
# The files contain the semantic version of the released information.  For
# example, latest-1.2.txt contains the version of the latest 1.2.x release.
function release::upload_version_data() {
  echo "+++ Uploading version data"
  local version_files="$(release::version_files ${VERSION})"
  for file in ${version_files}; do
    echo -e "+++\tProcessing file: ${file}"
    local semver_from_file="$(release::get_semver ${file})"
    echo -e "+++\t\tReleased  version: ${semver_from_file}"
    echo -e "+++\t\tRequested version: ${VERSION}"
    # TODO(filmil): Semver parsing library is broken.  Replace it with own
    # solution.  It does, however, work with clean releases.
    if semverGT "${VERSION}" "${semver_from_file}"; then
      local tempfile="$(mktemp --tmpdir semver.XXXXXX)"
      echo "${VERSION}" > "${tempfile}"
      local target="${RELEASE_BUCKET}/${file}"
      echo -e "+++\t\tUploading ${VERSION} to: ${target}"
      gsutil cp "${tempfile}" "${target}" 2>&1 | sed 's/^/--- /' -
      rm "${tempfile}"
    else
      echo -e "+++\t\tSkipping version update: ${file}"
    fi
  done
}

# Computes the artifacts of all files in OUTPUT_DIR.
function release::checksum_artifacts() {
  echo "+++ Checksumming artifacts"
  cd "${OUTPUT_DIR}"
  echo $PWD
  for file in "$(ls --ignore=*.md5 --ignore=*.sha1)"; do
    echo -e "+++ \tFile:\t${file}"
    md5sum -b "${file}" > "${file}.md5"
    sha1sum -b "${file}" > "${file}.sha1"
  done
  cd -
}

# Checks the global variables that represent input parameters to the releaser.
function release::check_parameters() {
  echo "+++ Checking parameters"
  if [[ "${STAGING_DIR}" == "" ]]; then
      echo "### Please specify STAGING_DIR"
      exit 1
  fi

  if [[ "${VERSION}" == "" ]]; then
      echo "### Please specify VERSION"
      exit 1
  fi

  if [[ "${RELEASE_BUCKET}" == "" ]]; then
      echo "### Please specify RELEASE_BUCKET"
      exit 1
  fi

  if [[ "${STABLE}" -ne "" ]]; then
      echo "+++ Releasing stable version"
  fi
}

function main() {
  release::check_parameters

  echo "+++ Using staging directory: ${STAGING_DIR}"
  echo "+++ Releasing version: ${VERSION}"
  OUTPUT_DIR="${STAGING_DIR}/out"
  mkdir -p "${OUTPUT_DIR}"
  INPUT_DIR="${STAGING_DIR}/in"
  mkdir -p "${INPUT_DIR}"

  release::stage_artifacts "${STAGING_DIR}/yaml/*" "${INPUT_DIR}/install/yaml"

  release::archive_artifacts "${INPUT_DIR}"
  release::checksum_artifacts
  release::upload_artifacts
  release::upload_version_data
}

main "$@"

