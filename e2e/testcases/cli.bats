#!/bin/bash

# Testcases related to nomos cli

set -euo pipefail

load "../lib/debug"
load "../lib/git"
load "../lib/namespaceconfig"
load "../lib/resource"
load "../lib/setup"
load "../lib/wait"

NOMOS_BIN=/opt/testing/go/bin/linux_amd64/nomos

FILE_NAME="$(basename "${BATS_TEST_FILENAME}" '.bats')"

setup() {
  setup::common
  setup::git::initialize
}

teardown() {
  setup::common_teardown
}

@test "${FILE_NAME}: CLI Vet Foo-corp" {
  ${NOMOS_BIN} vet --path=/opt/testing/e2e/examples/foo-corp-example/foo-corp
}

@test "${FILE_NAME}: CLI Vet Acme" {
  ${NOMOS_BIN} vet --path=/opt/testing/e2e/examples/acme
}

@test "${FILE_NAME}: CLI Init" {
  rm -rf "${BATS_TMPDIR}/empty-repo"
  mkdir "${BATS_TMPDIR}/empty-repo"
  cd "${BATS_TMPDIR}/empty-repo"

  git init
  ${NOMOS_BIN} init
  ${NOMOS_BIN} vet
}

@test "${FILE_NAME}: CLI Vet Multicluster" {
  debug::log "Expect default to fail since there is a collision in prod-cluster"
  wait::for -f -t 5 -- ${NOMOS_BIN} vet --path=/opt/testing/e2e/examples/parse-errors/cluster-specific-collision

  debug::log "Expect prod-cluster to fail since there is a collision"
  wait::for -f -t 5 -- ${NOMOS_BIN} vet --path=/opt/testing/e2e/examples/parse-errors/cluster-specific-collision --clusters=prod-cluster

  debug::log "Expect dev-cluster to succeed since there is no collision"
  ${NOMOS_BIN} vet --path=/opt/testing/e2e/examples/parse-errors/cluster-specific-collision --clusters=dev-cluster
}
