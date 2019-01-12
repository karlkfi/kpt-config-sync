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

# Builds error documentation to a piper client.
# Specifically, creates or reuses a piper client of the same name as the current branch.
#
# MUST be run from the root of the Nomos git repository.

DOCS_PATH="googledata/devsite/site-cloud/en/gke-on-prem/docs/wip/policy-management/docs/user/errors"

# Get name of current git branch.
BRANCH="$(git rev-parse --abbrev-ref HEAD)"

# Create piper client.
cd "$(p4 g4d -f "$BRANCH")" || exit
p4 client --set_option allwrite
G4_DIR="$(pwd)"
cd - || exit

# Write documentation to path.
.output/go/bin/linux_amd64/docgen --path="$G4_DIR/$DOCS_PATH"
cd - || exit

# Create a change for this. Automatically opens modified files.
# If there is an existing change, modifies it.
p4 change
