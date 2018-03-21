#!/bin/bash
#
# Copyright 2018 Nomos Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Common settings used by all the gce scripts.

# Directory for keeping the persistent Nomos state.  Ensure it exists and is
# a directory.
STOLOS_TMP=${STOLOS_TMP:-$HOME/stolos}
if [ ! -e ${STOLOS_TMP} ]; then
  mkdir ${STOLOS_TMP}
fi
if [ ! -d ${STOLOS_TMP} ]; then
  echo "File ${STOLOS_TMP} exists, but must be a directory."
  echo "Please either remove the file from that file path, or set $STOLOS_TMP"
  echo "to point to a path where you want us to store persistent state."
  exit 1
fi

# GCE configuration
export NUM_NODES=1
export KUBE_GCE_INSTANCE_PREFIX=${KUBE_GCE_INSTANCE_PREFIX:-${USER}}
