#!/bin/bash

set -euo pipefail

load "../lib/assert"
load "../lib/git"
load "../lib/setup"
load "../lib/wait"
load "../lib/resource"

setup() {
  setup::common
  setup::git::initialize

  local TEST_REPO_DIR=${BATS_TMPDIR}
  cd "${TEST_REPO_DIR}/repo"

  mkdir -p acme/system
  cp -r /opt/testing/e2e/examples/acme/system acme
  git add -A
}

teardown() {
  setup::common_teardown
}

readonly YAML_DIR=${BATS_TEST_DIRNAME}/../testdata

function stress::create_many_namespaces() {
  local num_namespaces
  num_namespaces="${1:-10}"

  local ns_file
  ns_file="$(mktemp)"
  for index in $(seq -w 1 "${num_namespaces}"); do
    cat <<EOF > "${ns_file}"
kind: Namespace
apiVersion: v1
metadata:
  name: ns-${index}
EOF
    git::add "${ns_file}" "acme/namespaces/ns-${index}/namespace.yaml"

  done

  debug::log "Commit all the namespaces"
  git::commit -m "Commit all the namespaces"

  debug::log "Check that namespaces have been created"
  for index in $(seq -w 1 "${num_namespaces}"); do
    # Namespaces are not created in order: this may fail if the earlier
    # namespaces are created very late in the process.
    wait::for -t 1800 -- kubectl get ns "ns-${index}"
    debug::log "Namespace ns-${index} exists"
  done

  rm "${ns_file}"
}

@test "Create 10 namespaces" {
  stress::create_many_namespaces "10"
}

@test "Create 100 namespaces" {
  stress::create_many_namespaces "100"
}

@test "Create 1000 namespaces" {
  stress::create_many_namespaces "1000"
}

