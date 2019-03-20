#!/bin/bash
#
# Copyright 2018 The Nomos Authors.
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

set -euo pipefail
set -x

# This remote branch needs to already exist.
# Manually create it if necessary.
REPO_VERSION=0.1.0

version=$(git describe --tags --always --dirty)
repo="/tmp/foo-corp-example"

rm -rf $repo
git clone -b $REPO_VERSION git@github.com:frankfarzan/foo-corp-example.git $repo
rsync -av --delete --exclude='.git' examples/foo-corp-example/ $repo
git -C $repo add .
git -C $repo commit -m "Synced from $version"
git -C $repo push origin HEAD:$REPO_VERSION
