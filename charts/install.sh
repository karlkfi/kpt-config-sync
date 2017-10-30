#!/bin/bash

set -euo pipefail

function delete() {
  releases="$(helm list -a)"
  if [[ "${releases}" != "" ]]; then
    release=$(echo "${releases}" | grep -v "^NAME" | cut -f 1)
    if [[ "${release}" != "" ]]; then
      helm delete --purge ${release}
    fi
  else
    echo "no release to delete"
  fi
}

function install() {
  project=$(gcloud config get-value project)
  namespace=stolos-system
  helm install ./stolos \
    --debug \
    --set registry=${project} \
    --namespace $namespace
}

for arg in "$@"; do
  case $arg in
    "delete")
      delete
      exit
      ;;
  esac
done

delete
install
