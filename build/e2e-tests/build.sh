#!/bin/bash

set -euo pipefail

quiet=false
tags=()
while (( $# != 0 )); do
  arg="${1:-}"
  shift
  case $arg in
    --quiet)
      quiet=true
      ;;

    -t)
      tags+=(-t "${1:-}")
      shift
      ;;

    *)
      echo "unknown arg $arg"
      exit 1
      ;;
  esac
done

docker build --target gcloud-install \
  -t e2e-tests-gcloud \
  --quiet=$quiet \
  "$(dirname "$0")"

exec docker build \
  --build-arg "DOCKER_GID=$(stat -c '%g' /var/run/docker.sock)" \
  --build-arg "UID=$(id -u)" \
  --build-arg "GID=$(id -g)" \
  --build-arg "UNAME=${USER}" \
  --quiet=$quiet \
  "$@" \
  "$(dirname "$0")"
