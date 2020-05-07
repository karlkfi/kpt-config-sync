#!/bin/bash

# Testcases related to nomos cli

set -euo pipefail

load "../lib/debug"
load "../lib/git"
load "../lib/namespaceconfig"
load "../lib/resource"
load "../lib/setup"
load "../lib/wait"
load "../lib/namespace"

NOMOS_BIN=/opt/testing/go/bin/linux_amd64/nomos

FILE_NAME="$(basename "${BATS_TEST_FILENAME}" '.bats')"

CMS_NS="config-management-system"
KS_NS="kube-system"

setup() {
  setup::common
  setup::git::initialize

  # Fixes a problem when running the tests locally
  if [[ -z "${KUBECONFIG+x}" ]]; then
    export KUBECONFIG="${HOME}/.kube/config"
  fi
}

teardown() {
  setup::common_teardown
}

@test "${FILE_NAME}: CLI bugreport.  Nomos running correctly." {
  if "${RAW_NOMOS}"; then
    # Trying to run bugreport in RAW_NOMOS mode causes an error
    # TODO (b/156016424): Decide whether this is a bug or a feature, and adjust this test accordingly.
    skip "Bugreport requires the configManagement CR to exist, so it doesn't work in RAW_NOMOS mode"
  fi

  # confirm all the pods are up
  if ! "${RAW_NOMOS}"; then
    namespace::check_exists ${KS_NS}
    resource::check_count -n ${KS_NS} -r pod -c 1 -l "k8s-app=config-management-operator"
  fi

  namespace::check_exists ${CMS_NS}
  resource::check_count -n ${CMS_NS} -r pod -c 1 -l "app=git-importer"
  resource::check_count -n ${CMS_NS} -r pod -c 1 -l "app=syncer"
  resource::check_count -n ${CMS_NS} -r pod -c 1 -l "app=monitor"

  "${NOMOS_BIN}" bugreport

  # check that zip exists
  BUG_REPORT_ZIP_NAME=$(get_bug_report_file_name)
  BUG_REPORT_DIR_NAME=${BUG_REPORT_ZIP_NAME%.zip}
  [[ -e "${BUG_REPORT_ZIP_NAME}" ]]

  /usr/bin/unzip "${BUG_REPORT_ZIP_NAME}"

  CurrentContext=$(kubectl config current-context)
  # Check that the correct files are there
  check_singleton "raw/${CurrentContext}/config-management-system/git-importer.*/git-sync.txt" "${BUG_REPORT_DIR_NAME}"
  check_singleton "raw/${CurrentContext}/config-management-system/git-importer.*/importer.txt" "${BUG_REPORT_DIR_NAME}"
  check_singleton "raw/${CurrentContext}/config-management-system/monitor.*/monitor.txt" "${BUG_REPORT_DIR_NAME}"
  check_singleton "raw/${CurrentContext}/config-management-system/syncer.*/syncer.txt" "${BUG_REPORT_DIR_NAME}"
  check_singleton "raw/${CurrentContext}/config-management-system/pods.txt" "${BUG_REPORT_DIR_NAME}"
  if ! "${RAW_NOMOS}"; then
    check_singleton "raw/${CurrentContext}/kube-system/config-management-operator.*/manager.txt" "${BUG_REPORT_DIR_NAME}"
  fi
  check_singleton "raw/${CurrentContext}/kube-system/pods.txt" "${BUG_REPORT_DIR_NAME}"
  check_singleton "processed/${CurrentContext}/version.txt" "${BUG_REPORT_DIR_NAME}"
  check_singleton "processed/${CurrentContext}/status.txt" "${BUG_REPORT_DIR_NAME}"
  check_singleton "raw/${CurrentContext}/cluster-scope/configmanagement/clusterconfigs.txt" "${BUG_REPORT_DIR_NAME}"
  check_singleton "raw/${CurrentContext}/cluster-scope/configmanagement/configmanagements.txt" "${BUG_REPORT_DIR_NAME}"
  check_singleton "raw/${CurrentContext}/cluster-scope/configmanagement/namespaceconfigs.txt" "${BUG_REPORT_DIR_NAME}"
  check_singleton "raw/${CurrentContext}/cluster-scope/configmanagement/repos.txt" "${BUG_REPORT_DIR_NAME}"
  check_singleton "raw/${CurrentContext}/cluster-scope/configmanagement/syncs.txt" "${BUG_REPORT_DIR_NAME}"
}

function get_bug_report_file_name {
  for f in ./*
  do
    if [[ -e $(echo "${f}" | grep "bug_report_.*.zip" ) ]]; then
      basename "${f}"
    fi
  done
}

function check_singleton {
  local regex_string="${1}"
  local base_dir="${2}"
  local directory_contents
  local num_results

  directory_contents="$(cd "${base_dir}"; find . -name "*.txt")"
  num_results="$(echo "${directory_contents}" | grep -c "${regex_string}")"

  if [[ "${num_results}" -eq 1 ]]; then
    return 0
  else
    printf "ERROR: %s results found matching regex: %s\n" "${num_results}" "${regex_string}"
    echo "${directory_contents}"

    return 1
  fi
}
