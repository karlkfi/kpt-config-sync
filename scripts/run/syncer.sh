#!/bin/bash

set -euo pipefail

cmd=$(basename ${0:-} .sh)
source $(dirname ${0:-})/common.sh

exec ${bin}/$(basename ${main}) \
  --logtostderr \
  -v=5 \
  "$@"
