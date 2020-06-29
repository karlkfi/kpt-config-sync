#!/bin/bash
#
# This testcase exists to verify that the "foo-corp" org is created fully.
#

set -euo pipefail

load "../lib/git"
load "../lib/namespace"
load "../lib/nomos"
load "../lib/resource"
load "../lib/setup"

FILE_NAME="$(basename "${BATS_TEST_FILENAME}" '.bats')"

test_setup() {
  setup::git::initialize
}

test_teardown() {
  setup::git::remove_all acme
}

test_-24-7bFILE-5fNAME-7d-3a_All_foo-2dcorp_created() { bats_test_begin "${FILE_NAME}: All foo-corp created" 24; 
  # TODO(frankf): POLICY_DIR is currently set to "acme" during installation.
  # This should be resolved with new repo format.
  git::add "${NOMOS_DIR}/examples/foo-corp-example/foo-corp" acme
  git::commit
  wait::for -t 180 -- nomos::repo_synced
}

bats_test_function test_-24-7bFILE-5fNAME-7d-3a_All_foo-2dcorp_created
