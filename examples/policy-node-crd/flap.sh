#!/bin/bash

set -x

while true; do
  kubectl create -f policyNodeOU1.yaml
  sleep 1
  kubectl delete policynodes/policy-node-ou1
  sleep 1
done
