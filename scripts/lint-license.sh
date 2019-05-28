#!/bin/bash

set -euo pipefail

echo "Running licenselinter: "
"${GOPATH}/bin/linux_amd64/licenselinter" -dir "$(pwd)" -print-aggregate > /tmp/LICENSE
echo "PASS"
echo

echo "Detecting LICENSE changes: "
diff -q /tmp/LICENSE LICENSE || (echo "LICENSE file has changed, commit the changes: 'cp .output/tmp/LICENSE LICENSE'" && exit 1)
echo "PASS"
echo
