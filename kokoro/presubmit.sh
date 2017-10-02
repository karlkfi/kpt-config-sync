#!/bin/bash

# Fail on any error.
set -e
# Display commands being run.
set -x

export GOPATH=/tmpfs/src/git/go
mkdir -p $GOPATH/src/github.com/google/stolos

# Copy our code over to github.com/stolos because that's the import path
cp -r git/stolos/* $GOPATH/src/github.com/google/stolos/

# Go get dependencies: Verbose, don't install, include test
go get -v -d -t ../...

# Test!
go test -v github.com/google/stolos/...
