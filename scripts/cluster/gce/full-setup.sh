#!/bin/bash

set -euo pipefail

echo "Setting project to stolos-dev"
gcloud config set project stolos-dev
echo "Setting compute zone to us-central1-b"
gcloud config set compute/zone us-central1-b

echo "Downloading kubernetes release"
./download-k8s-release.sh

echo "Setting up base kubernetes cluster"
./kube-up.sh

echo "Configuring monitoring"
./configure-monitoring.sh

echo "Configuring API server flags"
./configure-apiserver-for-stolos.sh
