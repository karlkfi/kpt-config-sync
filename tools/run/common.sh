#!/bin/bash

sync_dir=~/tmp/sync-dir
mkdir -p ${sync_dir}

cmd=$(basename ${0:-} .sh)

bin=$(echo ${GOPATH} | sed -e 's/.*://')/bin
repo=github.com/google/stolos
main=cmd/${cmd}

target=${repo}/${main}
echo "Building ${target}"
go install ${target}
