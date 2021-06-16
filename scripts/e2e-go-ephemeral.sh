#!/bin/bash

echo "+++ Building build/test-e2e-go/kind/Dockerfile prow-image"
docker build . -f build/test-e2e-go/kind/Dockerfile -t prow-image
# The .sock volume allows you to connect to the Docker daemon of the host.
# Part of the docker-in-docker pattern.

echo "+++ Running go e2e tests with" "$@"
docker run \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v "$ARTIFACTS":/logs/artifacts \
  --network="host" prow-image \
  ./build/test-e2e-go/e2e.sh "$@"
