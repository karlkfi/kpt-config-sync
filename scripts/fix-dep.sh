#!/bin/bash
#
# Due to the way we add licences to things dep tries to blow up all the added
# files and put the originals back.  This script attempts to automate cleaning
# up all the dep license changes.
#

# Fix deleted METADATA
git status | grep "deleted:.*METADATA" | sed -e 's/deleted://' | xargs git checkout

# Fix deleted LICENSE
git status | grep -E "deleted:.*(LICENSE|license)" | sed -e 's/deleted://' | xargs git checkout

# Fix modified LICENSE
git status | grep -E "modified:.*(LICENSE|license)" | sed -e 's/modified://' | xargs git checkout

# Remove newly vendored LICENSE.[ext]
git status | grep -E "vendor/.*(LICENSE|license)\\.{0,1}[a-z]*$" | xargs rm

# Not needed
rm -rf vendor/github.com/kubernetes-sigs/kubebuilder/docs/
rm -rf vendor/k8s.io/cluster-registry/docs/
