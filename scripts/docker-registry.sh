#!/bin/bash

set -euxo pipefail

# Starts a local docker registry and connects it to kind.
#
# Required for Go e2e tests to work. See e2e/doc.go.
#
# Installs kind if it is not already installed. If you already have kind
# installed but this does not work, make sure you're on version 0.10.0 or later.

# create registry container unless it already exists
reg_name='kind-registry'
reg_port='5000'
docker inspect "${reg_name}" &>/dev/null || (
  # The container doesn't exist.
  docker run \
    -d --restart=always -p "${reg_port}:5000" --name "${reg_name}" \
    registry:2
)

# The container exists, but might not be running.
# It's safe to run this even if the container is already running.
docker start "${reg_name}"

# Ensure kind v0.11.1 is installed.
# Dear future people: Feel free to upgrade this as new versions are released.
# Note that upgrading the kind version will require updating the image versions:
# https://github.com/kubernetes-sigs/kind/releases
kind &> /dev/null || (
  echo "Kind is not installed. Install v0.11.1."
  echo "https://kind.sigs.k8s.io/docs/user/quick-start/"
  exit 1
)

# Temporary.
# This allows the testing of kind v0.11.1 in CI without needing to update the
# prow kubekins-e2e image to use the new version of kind. Updating this image
# and setting this image in prow will cause all CI kind jobs to fail until this
# CL is merged in since the current ci kind jobs require kind v0.10.0.
if [ -n "$CI" ] && [ -n "$IMAGE" ] && [ "$IMAGE" == "gcr.io/k8s-testimages/kubekins-e2e:v20210808-1eaeec7-1.22" ]; then
  # Removes installed kind v0.10.0 from container
  rm -rf /bin/kind

  # Installs kind v0.11.1 at the same path as the kubekins-e2e dockerfile
  wget -q -O /bin/kind https://kind.sigs.k8s.io/dl/v0.11.1/kind-linux-amd64
  chmod +x /bin/kind
fi

kind version | grep v0.11.1 || (
  echo "Using unsupported kind version. Install v0.11.1."
  echo "https://kind.sigs.k8s.io/docs/user/quick-start/"
  exit 1
)

# Check if the "kind" docker network exists.
docker network inspect "kind" >/dev/null || (
  # kind doesn't create the docker network until it has been used to create a
  # cluster.
  kind create cluster
  kind delete cluster
)

# Connect the registry to the cluster network if it isn't already.
docker network inspect kind | grep "${reg_name}" || \
  docker network connect "kind" "${reg_name}"
