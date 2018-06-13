#!/bin/bash

load ../../lib/assert
load ../../lib/git
load ../../lib/namespace
load ../../lib/setup
load ../../lib/wait

# This cleans up any namespaces that were created by a testcase
function local_teardown() {
  echo kubectl delete ns -l "nomos.dev/testdata=true"
}

