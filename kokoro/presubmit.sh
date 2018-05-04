#!/bin/bash

# Fail on any error.
set -e
cd /tmpfs/src/git/stolos
cat kokoro/banner.txt
make test

