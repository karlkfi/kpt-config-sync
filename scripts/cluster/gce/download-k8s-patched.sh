#!/bin/bash

set -ev

rm -rf _kubernetes
git clone https://github.com/frankfarzan/kubernetes.git
mv kubernetes _kubernetes
cd _kubernetes
git checkout resolving_authz_webhook
make quick-release
