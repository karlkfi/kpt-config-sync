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

echo "+++ Building intermediate e2e image"
docker build --target gcloud-install \
  -t e2e-tests-gcloud \
  "$(dirname "$0")"

GID="${GID:-$(id -g)}"

echo "+++ Building e2e image"
exec docker build \
  --build-arg "UID=${UID}" \
  --build-arg "GID=${GID}" \
  --build-arg "UNAME=${USER}" \
  "${tags[@]}" \
  "$(dirname "$0")"
