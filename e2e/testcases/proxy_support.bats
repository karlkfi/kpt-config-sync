#!/bin/bash

load "../lib/setup"

set -euo pipefail

FILE_NAME="$(basename "${BATS_TEST_FILENAME}" '.bats')"

cms="config-management-system"

setup() {
  setup::common
  deploy_proxy
}

teardown() {
  remove_proxy
  setup::common_teardown
}

@test "${FILE_NAME}: Successfully syncs through a proxy" {
  skip "Proxy e2e test temporarily disabled until operator changes are rolled out and available to CI"
  kubectl apply -f "${BATS_TEST_DIRNAME}/../manifests/operator-config-git-proxy-enabled.yaml"

  token="$(get_token_at_HEAD)"
  wait::for -t 60 -- nomos::repo_synced --tokenOverride "${token}"

  # Check that the git-importer config map exists
  wait::for -s -t 60 -- [ -n "$(get_config_map_name)" ]

  # Verify that the variables show up in the configmap
  # We add this additional step to make sure that we aren't in a scenario where
  # the proxy isn't actually configured and the repo is thus able to sync
  # without testing the proxy functionality
  wait::for -s -t 60 -- verify_map_variable_exists "HTTP_PROXY"
}

verify_map_variable_exists () {
  var_name="${1}"
  kubectl get -n "${cms}" configmap "$(get_config_map_name)" -o yaml | grep "${var_name}"
  return $?
}

get_config_map_name () {
  kubectl -n "${cms}" get configmap | grep "git-importer" | awk '{ print $1 }'
}

deploy_proxy () {
  kubectl apply -f "${BATS_TEST_DIRNAME}/../manifests/tinyproxy.yaml"
}

remove_proxy () {
  kubectl -n "${cms}" delete deployment tinyproxy-deployment
  kubectl -n "${cms}" delete service tinyproxy-service
}

get_token_at_HEAD () {
  repo_url="https://github.com/GoogleCloudPlatform/csp-config-management.git"
  token_at_head="$(git ls-remote "${repo_url}" refs/heads/1.0.0 | awk '{ print $1 }')"
  echo "${token_at_head}"
}
