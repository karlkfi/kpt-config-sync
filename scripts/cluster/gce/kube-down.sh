#!/bin/bash

source $(dirname $0)/gce-common.sh

gcloud config get-value project
${NOMOS_TMP}/kubernetes/cluster/kube-down.sh
