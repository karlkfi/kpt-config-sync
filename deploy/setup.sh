#!/bin/bash
#
# Copyright 2017 Kubernetes Authors
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
#
# Setup script for installing pre-requisites before running other scripts in
# deploy.  Things in this file should generally follow the lifecycle of the
# kubernetes cluster and only be created once.
#

set -euo pipefail

REPO=$(dirname ${0:-})/..

function create_resource() {
  local cfg_file=${1:-}

  if ! kubectl get -f ${cfg_file} &> /dev/null; then
    kubectl create -f ${cfg_file}
  else
    echo "Already created: ${cfg_file}"
  fi
}

function delete_resource() {
  local cfg_file=${1:-}

  if kubectl get -f ${cfg_file} &> /dev/null; then
    kubectl delete -f ${cfg_file}
  else
    echo "Does not exist: ${cfg_file}"
  fi
}

# Create the stolos system namespace
namespace=${REPO}/manifests/namespace.yaml

# Create Custom Resource
policy_node_crd=${REPO}/manifests/policy-node-crd.yaml

# Syncer service account, role, rolebinding
service_account=${REPO}/manifests/service-account.yaml
role=${REPO}/manifests/cluster-role.yaml
rolebinding=${REPO}/manifests/cluster-rolebinding.yaml

ACTION=${ACTION:-create}
action_resource=${ACTION}_resource

case $ACTION in
  create|delete)
    echo "Will perform ${ACTION}"
    ;;
  *)
    echo "Invalid action ${ACTION}"
    exit 1
    ;;
esac

${action_resource} ${namespace}
sleep 2
${action_resource} ${policy_node_crd}
${action_resource} ${service_account}

if kubectl config current-context | grep "^gke_" &> /dev/null; then
  echo "On GKE, skipping setup for syncer ClusterRole/ClusteRoleBinding"
else
  ${action_resource} ${role}
  ${action_resource} ${rolebinding}
fi
