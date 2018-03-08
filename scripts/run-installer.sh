# !/bin/bash
#
# Copyright 2018 The Stolos Authors.
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

# Debugging.  Turn off for release.
set -x

VERSION=${VERSION:-latest}
CONFIG=${CONFIG:-configs/example.json}

readonly config_dir="$(dirname ${CONFIG})"
readonly config_filename="$(basename ${CONFIG})"

# The directory that is going to be used as output.
readonly tempdir=$(mktemp -d)

echo "+++ Using directory: ${tempdir}"
mkdir -p ${tempdir}/{kubeconfig,generated_certs,generated_configs}
docker run -it \
  -u "$(id -u):$(id -g)" \
  -v "${HOME}":/home/user \
  -v "${tempdir}/generated_certs":/opt/installer/generated_certs \
  -v "${tempdir}/kubeconfig":/opt/installer/kubeconfig \
  -v "${config_dir}":/opt/installer/configs \
  gcr.io/stolos-dev/installer:${VERSION} \
  bash -c "/opt/installer/entrypoint.sh --config=configs/${config_filename} $@"
echo "+++ Generated certs are in ${tempdir}"
