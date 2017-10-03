#!/bin/bash

# Fail on any error.
set -e

echo "============= Setting up STOLOS environment ========"
export GOPATH=/tmpfs/src/git/go
mkdir -p $GOPATH/src/github.com/google/stolos

# Copy our code over to github.com/stolos because that's the import path
cp -r git/stolos/* $GOPATH/src/github.com/google/stolos/

# Go get dependencies: Don't install, include test
echo "Go get ..."
go get -d -t ../...

echo "Go build ..."
go build github.com/google/stolos/...

echo "======================== TEST ======================"
go test -v github.com/google/stolos/...
