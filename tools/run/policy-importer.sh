#!/bin/bash

set -euo pipefail

cmd=$(basename ${0:-} .sh)
source $(dirname ${0:-})/common.sh

exec ${bin}/${cmd} \
  --logtostderr \
  -v=5 \
  --daemon \
  --sync_dir ${sync_dir}
