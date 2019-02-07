#!/bin/bash

# Testcases related to nomos cli

set -euo pipefail

load "../lib/debug"
load "../lib/git"
load "../lib/policynode"
load "../lib/resource"
load "../lib/setup"
load "../lib/wait"

NOMOS_BIN=/opt/testing/go/bin/linux_amd64/nomos

setup() {
  setup::common
  setup::git::initialize
}

@test "CLI Vet Foo-corp" {
  ${NOMOS_BIN} vet --path=/opt/testing/e2e/examples/foo-corp-example/foo-corp
}

@test "CLI Vet Acme" {
  ${NOMOS_BIN} vet --path=/opt/testing/e2e/examples/acme
}

@test "CLI Init" {
  rm -rf "${BATS_TMPDIR}/empty-repo"
  mkdir "${BATS_TMPDIR}/empty-repo"
  cd "${BATS_TMPDIR}/empty-repo"

  git init
  ${NOMOS_BIN} init
  ${NOMOS_BIN} vet
}
