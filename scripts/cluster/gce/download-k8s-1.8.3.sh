#!/bin/bash

set -ev

rm -rf _kubernetes
wget https://github.com/kubernetes/kubernetes/releases/download/v1.8.3/kubernetes.tar.gz -P /tmp
tar -xf /tmp/kubernetes.tar.gz
mv kubernetes _kubernetes
rm -rf /tmp/kubernetes.tar.gz
