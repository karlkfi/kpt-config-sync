#!/bin/bash

echo "+++ Building build/test-e2e-go/Dockerfile prow-image"
docker build . -f build/test-e2e-go/Dockerfile -t prow-image
# The .sock volume allows you to connect to the Docker daemon of the host.
# Part of the docker-in-docker pattern.

echo "+++ Running go e2e tests with" "$@"
docker run \
  -e GO111MODULE=on \
  -v /var/run/docker.sock:/var/run/docker.sock \
  --network="host" prow-image \
  go test ./e2e/... --e2e "$@"
