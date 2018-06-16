#!/bin/bash
# Tests for installer.sh

source "${BATS_TEST_DIRNAME}/installer.sh"

@test "installer::abspath relative pathname" {
  run installer::abspath foo
  [ "$output" = "/go/src/github.com/google/nomos/foo" ]
}

@test "installer::abspath absolute pathname" {
  run installer::abspath /foo
  [ "$output" = "/foo" ]
}

@test "installer::abspath wonky pathname" {
  run installer::abspath foo/../bar
  [ "$output" = "/go/src/github.com/google/nomos/foo/../bar" ]
}

@test "installer::bool" {
  run installer::bool off
  [ "$output" = "false" ]
  run installer::bool on
  [ "$output" = "true" ]
}

