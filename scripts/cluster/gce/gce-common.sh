#!/bin/bash

# Directory for keeping the kubernetes source
STOLOS_TMP=${STOLOS_TMP:-$HOME/stolos}

# GCE configuration
export NUM_NODES=1
export KUBE_GCE_INSTANCE_PREFIX=${KUBE_GCE_INSTANCE_PREFIX:-${USER}}
