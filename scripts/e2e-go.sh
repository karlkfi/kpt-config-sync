#!/bin/bash

echo "+++ Running Mono Repo tests"
GO111MODULE=on go test ./e2e/... --e2e "$@"
mono_repo_exit=$?

echo "+++ Running Multi Repo tests"
GO111MODULE=on go test ./e2e/... --e2e --multirepo "$@"
multi_repo_exit=$?

if (( mono_repo_exit != 0 || multi_repo_exit != 0 )); then
  exit 1
fi
