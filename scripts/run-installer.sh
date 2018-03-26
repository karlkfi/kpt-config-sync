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

# For debugging.  Remove before release.
set -x

##### HEADER #####

# This script executes the nomos installer using a docker runtime.
#
# --interactive  Use the installer interactively. If not specified batch
#                mode is used. When batch mode is used --config is required
#                default: false
# --config       The configuration file to use in batch mode. To create an
#                installer yaml, do an interactive install and copy the
#                generated yaml from $PWD/gen_configs/generated.yaml to a
#                preferred location and
#                default: ""
# --version      Specify the gcr.io tag to use for each nomos component's
#                docker container
#                default: '${VERSION:-"latest"}'
# --container    Specify the gcr.io container repo to use to source each
#                nomos component's docker container.
#                default: gcr.io/projects/stolos-dev
# --output_dir   The directory to pickup the generated configuration including
#                certs, kubeconfigs, yamls, etc.
#                default: $PWD

##### CLI Arguments #####

usage()
{
    cat << E0
      Usage: $0 [options]
        --help
        --version=<version>
        --output_dir=<output_dir>
        --interactive=<interactive>
        --container=<container>
        --config=<config>
E0
}

# Reset in case getopts has been used previously in the shell.
OPTIND=1

# The installer container.  The named registry should be publicly accessible.
INSTALLER_CONTAINER="gcr.io/nomos-release/installer"
CONFIG=""
# If interactive is set to a nonempty string, the installer will use a
# menu-driven interactive installer.
INTERACTIVE=""
OUTPUT_DIR=$(pwd)
# The semantic verison number of the release to install.
VERSION=${VERSION:-"latest"}

getopt --test > /dev/null
if [[ $? -ne 4 ]]; then
    echo "I’m sorry, `getopt --test` failed in this environment."
    exit 1
fi

OPTIONS=hvoitc
LONGOPTIONS=help::,version::,output_dir::,interactive::,container::,config::

# -temporarily store output to be able to check for errors
# -e.g. use “--options” parameter by name to activate quoting/enhanced mode
# -pass arguments only via   -- "$@"   to separate them correctly
PARSED=$(getopt -s bash --options=$OPTIONS --longoptions=$LONGOPTIONS --name "$0" -- "$@")
if [[ $? -ne 0 ]]; then
    # e.g. $? == 1
    #  then getopt has complained about wrong arguments to stdout
    exit 2
fi
# read getopt’s output this way to handle the quoting right:
eval set -- "$PARSED"

# now enjoy the options in order and nicely split until we see --
while true; do
    case "$1" in
        -h|--help)
            usage
            exit 0
            shift
            ;;
        -v|--version)
            VERSION="$2"
            shift 2
            ;;
        -t|--container)
            INSTALLER_CONTAINER="$2"
            shift 2
            ;;
        -c|--config)
            CONFIG="$2"
            shift 2
            ;;
        -o|--output_dir)
            OUTPUT_DIR="$2"
            shift 2
            ;;
        -i|--interactive)
            INTERACTIVE=1
            shift 2
            ;;
        --)
            shift
            break
            ;;
        *)
            echo "invalid option config"
            exit 3
            ;;
    esac
done

## Main ##

readonly gcloud_prg="$(which gcloud)"
if [[ -z "${gcloud_prg}" ]]; then
  echo "gcloud is required."
  exit 1
fi

CONFIG_DIR="examples"
CONFIG_BASENAME="quickstart.yaml"
if [[ -z "${CONFIG}" ]]; then
  CONTAINER_CONFIG_FILE="examples/quickstart.yaml"
else
  # $CONFIG specified, ensure that it's an existing file, and map it into the
  # container.
  CONFIG="$(realpath $CONFIG)"
  if ! [[ -f "${CONFIG}" ]]; then
    echo "file not found: ${CONFIG}"
    exit 1
  fi
  CONFIG_DIR="$(dirname $CONFIG)"
  CONFIG_BASENAME="$(basename $CONFIG)"
  CONTAINER_CONFIG_FILE="configs/${CONFIG_BASENAME}"
fi

# Only suggest the user where there is a gcloud utility available.
readonly suggested_user="$(gcloud config get-value account)"

# Temporarily, pull the latest image to the local docker daemon cache.
# TODO(filmil): Remove this gimmick once we have a public cluster registry
# to pull from.
gcloud docker -- pull "${INSTALLER_CONTAINER}:${VERSION}"

# TODO(filmil): Move the environment variables to flags.
echo "+++ Using output directory: ${OUTPUT_DIR}"
mkdir -p ${OUTPUT_DIR}/{kubeconfig,certs,gen_configs,logs}
docker run -it \
  -u "$(id -u):$(id -g)" \
  -v "${HOME}":/home/user \
  -v "${OUTPUT_DIR}/certs":/opt/installer/certs \
  -v "${OUTPUT_DIR}/gen_configs":/opt/installer/gen_configs \
  -v "${OUTPUT_DIR}/kubeconfig":/opt/installer/kubeconfig \
  -v "${OUTPUT_DIR}/logs":/tmp \
  -v "${CONFIG_DIR}":/opt/installer/configs \
  -e "INTERACTIVE=${INTERACTIVE}" \
  -e "VERSION=${VERSION}" \
  -e "CONFIG=${CONTAINER_CONFIG_FILE}" \
  -e "CONFIG_OUT=gen_configs/generated.yaml" \
  -e "SUGGESTED_USER=${suggested_user}" \
  "${INSTALLER_CONTAINER}:${VERSION}" "$@"
echo "+++ Generated files are available in ${OUTPUT_DIR}"

