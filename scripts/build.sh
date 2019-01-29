#!/bin/bash
#
# Copyright 2016 The Kubernetes Authors.
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

PKG=""
VERSION=""
PLATFORM=""
BUILD_MODE=""

while [[ $# -gt 0 ]]; do
  ARG="${1:-}"
  shift
  case ${ARG} in
    --pkg)
      PKG=${1:-}
      shift
      ;;
    --version)
      VERSION=${1:-}
      shift
      ;;
    --platform)
      PLATFORM=${1:-}
      shift
      ;;
    --build_mode)
      BUILD_MODE=${1:-}
      shift
      ;;
    --)
      break
      ;;
    *)
      echo "Invalid arg: ${ARG}"
      exit 1
    ;;
  esac
done

if [ -z "${PKG}" ]; then
    echo "PKG must be set"
    exit 1
fi
if [ -z "${VERSION}" ]; then
    echo "VERSION must be set"
    exit 1
fi
# Platform format: $GOOS-$GOARCH (e.g. linux-amd64).
if [ -z "${PLATFORM}" ]; then
    echo "PLATFORM must be set"
    exit 1
fi
if [ -z "${BUILD_MODE}" ]; then
    echo "BUILD_MODE must be set"
    exit 1
fi

# Example: "linux-amd64" -> ["linux" "amd64"]
IFS='-' read -r -a platform_split <<< "${PLATFORM}"
GOOS="${platform_split[0]}"
GOARCH="${platform_split[1]}"

env GOOS="${GOOS}" GOARCH="${GOARCH}" CGO_ENABLED="0" go install             \
    -installsuffix "static"                                        \
    -ldflags "-X ${PKG}/pkg/version.VERSION=${VERSION}"            \
    "$@"

output_dir="${GOPATH}/bin/${GOOS}_${GOARCH}"
mkdir -p "${output_dir}"
find "${GOPATH}/bin" -maxdepth 1 -type f -exec mv {} "${output_dir}" \;

# Use upx to reduce binary size.
if [ "${BUILD_MODE}" = "release" ]; then
  upx "${output_dir}/*"
fi
