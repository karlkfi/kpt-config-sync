#!/bin/sh

# Distributes the files
# Root permission is needed for this.
set -v

cp authz.yaml /etc/srv/kubernetes/authz.yaml
cp ca-webhook.crt /etc/srv/kubernetes/ca-webhook.crt
cp kube-apiserver.manifest.patched \
  /etc/kubernetes/manifests/kube-apiserver.manifest

