#!/bin/bash

set -euo pipefail

load "../lib/stress"
load "../lib/setup"

FILE_NAME="$(basename "${BATS_TEST_FILENAME}" '.bats')"

setup() {
  setup::common
  setup::git::initialize

  local TEST_REPO_DIR=${BATS_TMPDIR}
  cd "${TEST_REPO_DIR}/repo"

  mkdir -p acme/system
  cp -r "${NOMOS_DIR}/examples/acme/system" acme
  git add -A
}

teardown() {
  setup::common_teardown
}

test_-24-7bFILE-5fNAME-7d-3a_Create_10_namespaces() { bats_test_begin "${FILE_NAME}: Create 10 namespaces" 26; 
  stress::create_many_namespaces "10"
}

test_-24-7bFILE-5fNAME-7d-3a_Create_100_namespaces() { bats_test_begin "${FILE_NAME}: Create 100 namespaces" 30; 
  stress::create_many_namespaces "100"
}

test_-24-7bFILE-5fNAME-7d-3a_Create_1000_namespaces() { bats_test_begin "${FILE_NAME}: Create 1000 namespaces" 34; 
  stress::create_many_namespaces "1000"
}

test_-24-7bFILE-5fNAME-7d-3a_Create_10_resources_in_a_single_namespace() { bats_test_begin "${FILE_NAME}: Create 10 resources in a single namespace" 38; 
  stress::create_many_resources "10"
}

test_-24-7bFILE-5fNAME-7d-3a_Create_100_resources_in_a_single_namespace() { bats_test_begin "${FILE_NAME}: Create 100 resources in a single namespace" 42; 
  stress::create_many_resources "100"
}

test_-24-7bFILE-5fNAME-7d-3a_Create_500_resources_in_a_single_namespace() { bats_test_begin "${FILE_NAME}: Create 500 resources in a single namespace" 46; 
  stress::create_many_resources "500"
}

test_-24-7bFILE-5fNAME-7d-3a_Create_100_resources-2c_slow_speed() { bats_test_begin "${FILE_NAME}: Create 100 resources, slow speed" 50; 
  stress::create_many_commits "100" "10"
}

test_-24-7bFILE-5fNAME-7d-3a_Create_100_resources-2c_medium_speed() { bats_test_begin "${FILE_NAME}: Create 100 resources, medium speed" 54; 
  stress::create_many_commits "100" "3"
}

test_-24-7bFILE-5fNAME-7d-3a_Create_100_resources-2c_fast() { bats_test_begin "${FILE_NAME}: Create 100 resources, fast" 58; 
  stress::create_many_commits "100" "1"
}

test_-24-7bFILE-5fNAME-7d-3a_Create_100_resources-2c_RLY_fast() { bats_test_begin "${FILE_NAME}: Create 100 resources, RLY fast" 62; 
  stress::create_many_commits "100" "0"
}

bats_test_function test_-24-7bFILE-5fNAME-7d-3a_Create_10_namespaces
bats_test_function test_-24-7bFILE-5fNAME-7d-3a_Create_100_namespaces
bats_test_function test_-24-7bFILE-5fNAME-7d-3a_Create_1000_namespaces
bats_test_function test_-24-7bFILE-5fNAME-7d-3a_Create_10_resources_in_a_single_namespace
bats_test_function test_-24-7bFILE-5fNAME-7d-3a_Create_100_resources_in_a_single_namespace
bats_test_function test_-24-7bFILE-5fNAME-7d-3a_Create_500_resources_in_a_single_namespace
bats_test_function test_-24-7bFILE-5fNAME-7d-3a_Create_100_resources-2c_slow_speed
bats_test_function test_-24-7bFILE-5fNAME-7d-3a_Create_100_resources-2c_medium_speed
bats_test_function test_-24-7bFILE-5fNAME-7d-3a_Create_100_resources-2c_fast
bats_test_function test_-24-7bFILE-5fNAME-7d-3a_Create_100_resources-2c_RLY_fast
