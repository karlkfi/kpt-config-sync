#!/bin/bash

# Fail on any error.
set -e

echo "============= Setting up STOLOS environment ========"
export GOPATH=/tmpfs/src/git/go
STOLOS_DIR=$GOPATH/src/github.com/google/stolos
mkdir -p $STOLOS_DIR

# Copy our code over to github.com/stolos because that's the import path
cp -r git/stolos/* $STOLOS_DIR/

cd $STOLOS_DIR

# Go get dependencies: Don't install, include test
echo "Go get ..."
go get -d -t ../...

echo "======================== BUILD ====================="
make build-all

echo "======================== TEST ======================"
make test-all

