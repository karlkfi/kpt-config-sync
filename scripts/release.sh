#!/bin/bash
#
# Release helper script.  This tool will be used by nomos-oncall engineers when
# doing things described in go/nomos-release.
#

usage() {
  cat <<EOF
Usage: $0 [OPTIONS]

Options
    -a action [REQUIRED]
       One of the following values:
         staging: promote nomos binaries and operator to staging
         staging-verified: promote from staging to staging-verified
         stable: promote from staging-verified to stable
         clean-staging: remove nomos binaries and operator in staging
         restore: replace contents of stable with a specific release
         versions: display versions in staging, staging-verified, and stable
    -n nomos_release_version
       REQUIRED for 'staging' and 'restore'
       Everything after the 'v' in the bucket name,  e.g. 0.1.2-rc.34
    -o operator_release_version
       REQUIRED for 'staging' and 'restore'
       Everything after the 'nomos-operator-v' in the bucket name,
       e.g. 0.1.2-rc.34
    -p project - use a different project from the default
       (default=$DEFAULT_PROJECT)
    -b bucket - use a different top-level bucket from the default
       (default=$DEFAULT_BUCKET).
       Note: must include gs://.

Examples

    # stage nomos v0.1.2-rc.34 and nomos-operator-v9.8.7-rc.65
    release.sh -a staging -n 0.1.2-rc.34 -o 9.8.7-rc.65

    # promote whatever's in staging to staging-verified (both nomos and
    # nomos-operator)
    release.sh -a staging-verified

    # promote whatever's in staging-verified to stable (both nomos and
    # nomos-operator)
    release.sh -a stable

    # wipe out the contents of staging and operator-staging on the project
    # "sandyjensen-playground" in GCS bucket "gs://nomos-testing"
    release.sh -a clean-staging -p sandyjensen-playground -b nomos-testing

    # replace whatever's in stable and operator-stable with stable-0.1.2-rc.34
    # and operator-stable-9.8.7-rc.65
    release.sh -a restore -n 0.1.2-rc.34 -o 9.8.7-rc.65

    # dump out the versions of whatever's in
    # stable, staging, staging-verified, operator-stable, operator-staging,
    # and operator-staging-verified
    release.sh -a versions

EOF
    exit 1
}

set -euo pipefail

readonly DEFAULT_PROJECT=config-management-release
readonly DEFAULT_BUCKET=gs://config-management-release

# We will allow the user to overwrite the values of PROJECT and BUCKET from the
# command line (as -p and -b flags respectively).
PROJECT=$DEFAULT_PROJECT
BUCKET=$DEFAULT_BUCKET

while getopts "a:n:o:p:b:" arg; do
  case $arg in
    a)
      ACTION="${OPTARG}"
      { \
        [ "$ACTION" = "staging" ]          || \
        [ "$ACTION" = "staging-verified" ] || \
        [ "$ACTION" = "stable" ]           || \
        [ "$ACTION" = "clean-staging" ]    || \
        [ "$ACTION" = "restore" ]          || \
        [ "$ACTION" = "versions" ];           \
      } || usage
      ;;
    n)
      NOMOS_VERSION="${OPTARG}"
      ;;
    o)
      OPERATOR_VERSION="${OPTARG}"
      ;;
    p)
      PROJECT="${OPTARG}"
      ;;
    b)
      BUCKET="${OPTARG}"
      ;;
    *)
      usage
      ;;
  esac
done

# This contortion satisfies the linter:
# https://github.com/koalaman/shellcheck/wiki/SC1026
for a in "staging" "restore"; do
  if [ "$ACTION" = "$a" ]; then
    if [ "${NOMOS_VERSION-}" = "" ]; then
      echo "-n nomos_version required for $ACTION"
      exit 1
    fi
    if [ "${OPERATOR_VERSION-}" = "" ]; then
      echo "-o operator_version required for $ACTION"
      exit 1
    fi
  fi
done

readonly restore_project="$(gcloud config get-value project)"
restore() {
  gcloud config set project "$restore_project" >/dev/null 2>&1
}
gcloud config set project "$PROJECT" >/dev/null 2>&1
trap 'restore' ERR
trap 'restore' EXIT

# We'll use each of these commands several times in the exact same way;
# shortening them here for convenience/readability.
readonly rsync="gsutil -m rsync -r -d -p "
readonly setmeta="gsutil -m setmeta -r -h "
readonly rm="gsutil -m rm -r "
readonly setacl="gsutil acl ch -r -u AllUsers:R "

# get_version echoes the RC version for the requested bucket to STDOUT, where it
# may be picked up by the function caller.
#
# Args:
#   $1: GCS bucket to query for metadata (required)
#   $2: Header name where the metadata is stored (required)
#
# We store the RC version as GCS bucket & object metadata when we promote an RC
# to staging.
#
# gsutil doesn't have a nice mechanism for getting at bucket metadata.  `gsutil
# ls -L` is required to get at the information, but the information is
# unstructured, so we're just grepping for the label (different for nomos and
# the nomos-operator).  gsutil also doesn't seem to have a way to restrict ls to
# the bucket, so we get a value returned for every object contained by the
# bucket, and use head -1 to ignore all but the first.
function get_version {
  local this_bucket=$1
  local this_label=$2

  gsutil ls -L "$this_bucket" | \
    grep "$this_label" | \
    head -1 | \
    cut -f2 -d: | \
    sed -e 's/^[[:space:]]*//'
}

# bucket_exists determines whether a particular GCS bucket exists.
#
# Args:
#   $1: GCS bucket to test
#
# gsutil has no good way to determine whether/not a bucket exists,
# this ugly hack provides that function.
function bucket_exists {
  local this_bucket=$1
  [[ $(gsutil ls -b "$this_bucket" 2>/dev/null | wc -l) -gt 0 ]]
}

# rm_if_exists will `gsutil -m rm -r` the requested bucket iff it exists.
#
# Args:
#   $1: GCS bucket to remove
#
# This prevents the spam to STDOUT/STDERR that would result if we tried to
# delete a non-existent bucket.
function rm_if_exists {
  local this_bucket=$1

  if bucket_exists "$this_bucket"; then
    $rm "$this_bucket"
  fi
}

readonly nomos_rc_bucket="$BUCKET"
readonly nomos_rc_prefix="v"
readonly nomos_staging_bucket="${BUCKET}/staging"
readonly nomos_staging_verified_bucket="${BUCKET}/staging-verified"
readonly nomos_stable_bucket="${BUCKET}/stable"
readonly nomos_archive_bucket_base="${BUCKET}/stable-v"
readonly nomos_version_label="nomos-version"

readonly operator_rc_bucket="${BUCKET}/operator-rc"
readonly operator_rc_prefix="nomos-operator-v"
readonly operator_staging_bucket="${BUCKET}/operator-staging"
readonly operator_staging_verified_bucket="${BUCKET}/operator-staging-verified"
readonly operator_stable_bucket="${BUCKET}/operator-stable"
readonly operator_archive_bucket_base="${BUCKET}/operator-stable-v"
readonly operator_version_label="nomos-operator-version"

cat <<EOF
project: $PROJECT
bucket: $BUCKET
EOF
[[ "${NOMOS_VERSION-}" ]] && echo "nomos_version: ${NOMOS_VERSION}"
[[ "${OPERATOR_VERSION-}" ]] && echo "operator_version: ${OPERATOR_VERSION}"

case "${ACTION}" in

  # The staging action will (1) remove existing staging and operator-staging
  # buckets, (2) sync the user-specified release candidates to the staging and
  # operator-staging buckets, then set a metadata label on the new staging GCS
  # buckets so that we can figure out the RC versions in the future.
  staging)
    rm_if_exists "${nomos_staging_bucket}"
    rm_if_exists "${operator_staging_bucket}"

    $rsync "${nomos_rc_bucket}/${nomos_rc_prefix}${NOMOS_VERSION}" \
      "${nomos_staging_bucket}"
    $setmeta "x-goog-meta-${nomos_version_label}:${NOMOS_VERSION}" \
      "${nomos_staging_bucket}"

    $rsync "${operator_rc_bucket}/${operator_rc_prefix}${OPERATOR_VERSION}" \
      "${operator_staging_bucket}"
    $setmeta "x-goog-meta-${operator_version_label}:${OPERATOR_VERSION}" \
      "${operator_staging_bucket}"
    ;;

  # The staging-verified action will (1) remove existing staging-verified and
  # operator-staging-verified buckets, then (2) copy the current staging and
  # operator-staging buckets into the staging-verified and
  # operator-staging-verified buckets.  The metadata that was set in the staging
  # action will be copied automatically to the staging-verified buckets.
  staging-verified)
    rm_if_exists "${nomos_staging_verified_bucket}"
    rm_if_exists "${operator_staging_verified_bucket}"
    $rsync "${nomos_staging_bucket}" "${nomos_staging_verified_bucket}"
    $rsync "${operator_staging_bucket}" "${operator_staging_verified_bucket}"
    ;;

  # The stable action will (1) copy current versions of stable and
  # operator-stable to backup locations.  The backup location names will be in
  # the form stable-<version> and operator-stable-<version>, with <version>
  # extracted from the metadata labels that were set when the RCs were copied to
  # staging.  Next, (2) existing stable and operator-stable buckets will be
  # replaced with the contents of the staging-verified and
  # operator-staging-verified buckets.  Finally, (2) the new stable and
  # operator-stable buckets are made world-readable.
  stable)
    # Stable directory might not exist, though this should be rare in
    # production.
    if bucket_exists "${nomos_stable_bucket}"; then
      NOMOS_VERSION=$(get_version \
        "${nomos_stable_bucket}" "${nomos_version_label}")
      $rsync "${nomos_stable_bucket}" "${nomos_archive_bucket_base}${NOMOS_VERSION}"
    fi

    if bucket_exists "${operator_stable_bucket}"; then
      OPERATOR_VERSION=$(get_version \
        "${operator_stable_bucket}" "${operator_version_label}")
      $rsync "${operator_stable_bucket}" \
        "${operator_archive_bucket_base}${OPERATOR_VERSION}"
    fi

    rm_if_exists "${nomos_stable_bucket}"
    rm_if_exists "${operator_stable_bucket}"
    $rsync "${nomos_staging_verified_bucket}" "${nomos_stable_bucket}"
    $rsync "${operator_staging_verified_bucket}" "${operator_stable_bucket}"

    $setacl "${nomos_stable_bucket}"
    $setacl "${operator_stable_bucket}"
    ;;

  # The clean-staging action is a convenience function, to be used if a
  # staging/operator-staging version are considered unsuitable (failed antfood,
  # failed fishfood).  It simply removes the staging and operator-staging
  # directories.
  clean-staging)
    rm_if_exists "${nomos_staging_bucket}"
    rm_if_exists "${operator_staging_bucket}"
    ;;

  # The restore action will replace the contents of the stable and
  # operator-stable buckets with the user-specified previous stable versions.
  # First, it verifies that the versions the user wants to restore exist.  Then
  # (2) it removes the current stable and operator-stable buckets, (3) replaces
  # them with the user-specified versions, and (4) updates the acls on the new
  # stable versions to make them world-readable.  The previous stable versions
  # are left untouched.
  restore)
    target_nomos_restore="${nomos_archive_bucket_base}${NOMOS_VERSION}"
    target_operator_restore="${operator_archive_bucket_base}${OPERATOR_VERSION}"

    if ! bucket_exists "$target_nomos_restore"; then
      echo "$target_nomos_restore does not exist, cannot proceed"
      exit 2
    fi

    if ! bucket_exists "$target_operator_restore"; then
      echo "$target_operator_restore does not exist, cannot proceed"
      exit 2
    fi

    rm_if_exists "${nomos_stable_bucket}"
    rm_if_exists "${operator_stable_bucket}"

    $rsync "$target_nomos_restore" "${nomos_stable_bucket}"
    $rsync "$target_operator_restore" "${operator_stable_bucket}"

    $setacl "${nomos_stable_bucket}"
    $setacl "${operator_stable_bucket}"
    ;;

  # The versions action will print the bucket location and versions of the
  # staging, staging-verified, stable, operator-staging,
  # operator-staging-verified, and operator-stable buckets.
  versions)
    for dir in \
      "${nomos_staging_bucket}" \
      "${nomos_staging_verified_bucket}" \
      "${nomos_stable_bucket}"; do
      this_version=$(get_version "${dir}" "${nomos_version_label}")
      echo "${dir}: ${this_version}"
    done
    for dir in \
      "${operator_staging_bucket}" \
      "${operator_staging_verified_bucket}" \
      "${operator_stable_bucket}"; do
      this_version=$(get_version "${dir}" "${operator_version_label}")
      echo "${dir}: ${this_version}"
    done
    ;;

esac
