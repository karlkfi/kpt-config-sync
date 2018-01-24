#!/bin/bash
set -x
kubectl delete ValidatingWebhookConfiguration stolos-resource-quota --ignore-not-found
kubectl delete ns stolos-system

