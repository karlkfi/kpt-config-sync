#!/bin/bash

source $(dirname $0)/gce-common.sh

gcloud config get-value project
${STOLOS_TMP}/kubernetes/cluster/kube-up.sh
