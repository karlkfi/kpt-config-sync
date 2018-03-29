#!/bin/bash

# Fail on any error.
set -e

echo "============= Setting up NOMOS environment ========"
export GOPATH=/tmpfs/src/git/go
NOMOS_DIR=$GOPATH/src/github.com/google/nomos
mkdir -p $NOMOS_DIR

# Copy our code over to github.com/nomos because that's the import path
git clone git/stolos $NOMOS_DIR

cd $NOMOS_DIR

# Go get dependencies: Don't install, include test
echo "Go get ..."
go get -d -t ../...

echo "======================== BUILD ====================="
make DOCKER_INTERACTIVE="" build

echo "======================== TEST ======================"
make DOCKER_INTERACTIVE="" test

