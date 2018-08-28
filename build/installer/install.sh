#!/bin/bash
#
# Copyright 2018 The Nomos Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License")
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

echo "+++ Running installer"

# Only suggest the user where there is a gcloud utility available.
suggested_user=""
if command -v gcloud &> /dev/null ; then
   suggested_user="$(gcloud config get-value account)"
fi

# Guess which operating system is used.  Work is underway to support more
# OS/architecture combinations.
readonly os="$(uname)"
readonly arch="$(uname -m)"
dir=""
case "${os}" in
  Linux)
    case "${arch}" in
      x86_64)
        dir="linux_amd64"
      ;;
      *)
        echo "### Unsupported architecture: ${arch}"
        exit 1
      ;;
    esac
  ;;

  Darwin)
    dir="darwin_amd64"
  ;;

  *)
    echo "### Unsupported operating system: ${os}"
    exit 1
  ;;
esac

INSTALLER_ARGS=(
  --alsologtostderr
  --log_dir=/tmp
  --suggested_user="${suggested_user}"
  --vmodule=main=5,configgen=3,kubectl=3,installer=3,exec=3
  --work_dir="${PWD}"
)
./bin/"${dir}"/installer "${INSTALLER_ARGS[@]}" "$@"

