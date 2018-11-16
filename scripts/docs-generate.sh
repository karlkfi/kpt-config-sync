#!/bin/bash
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

# Builds documentation hermetically.

set -x

mkdir -p "${HOME}/.grip"
cat > "${HOME}/.grip/settings.py" <<EOF
CACHE_DIRECTORY = '/go/src/${REPO}/.output/staging/docs/asset'
EOF

# Convert markdown to HTML.
find "${DOCS_STAGING_DIR}" -name "*.md" \
  -exec grip --export {} --no-inline \; \
  -exec rm {} \;

if grep "GitHub rate limit reached" -r "${DOCS_STAGING_DIR}"; then
  echo "Github rate limit reached"
  exit 1
fi

# Post-process HTML.
# 1. Convert links to our docs to be relative.
# 2. Convert links to reflect flattened directory.
# 3. Convert links to our docs to use html suffix.

# Warning SC1117 is too eager for the line below.
# shellcheck disable=SC1117
find "${DOCS_STAGING_DIR}" -name "*.html" \
  -exec sed -i -r "s:/__/grip/::g" {} \;  \
  -exec sed -i -r "/http/b; s:\.md:\.html:g" {} \;

# 4. Update path to asset dir for docs in subdir.
find "${DOCS_STAGING_DIR}/docs" -name "*.html" \
  -exec sed -i -r "s:\"asset/:\"../../asset/:g" {} \;
