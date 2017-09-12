#!/bin/bash

set -euo pipefail

# TODO: get this from yaml file or create yaml file from template
sa_name=namespace-syncer-robot

sa_cfg=./serviceaccount.yaml

if ! kubectl get sa/${sa_name} &> /dev/null; then
  kubectl create -f ./${sa_cfg}
else
  echo "Account ${sa_name} exists, will not create"
fi

secret_name=$(kubectl get sa/${sa_name} -o=jsonpath='{.secrets[0].name}')
echo "Account has secret ${secret_name}"

kubectl get secrets/${secret_name} -oyaml > secret.yaml
echo "Secret in secret.yaml"
