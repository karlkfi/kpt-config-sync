#!/bin/bash

set -euo pipefail

tmp=$(mktemp -d)

# This regex matches license files found by https://github.com/google/go-licenses.
licenseRegex="\(LICEN\(S\|C\)E\|COPYING\|README\|NOTICE\)"

# Find all license files in vendor/
# Note that packages may contain multiple license, or may have both a LICENSE
# and a NOTICE which must BOTH be distributed with the code.
licenses="${tmp}"/licenses.txt

# Ensure the vendor directory only includes the project's build dependencies.
go mod tidy
go mod vendor
find vendor/ -regex ".*/${licenseRegex}" > "${licenses}"
sort "${licenses}" -dufo "${licenses}"

# Default to LICENSE as the output file, but allow overriding it.
out="LICENSE"
if [[ $# -gt 0 ]]; then
  out=$1
fi

# Preamble at beginning of LICENSE.
echo "THE FOLLOWING SETS FORTH ATTRIBUTION NOTICES FOR THIRD PARTY SOFTWARE THAT MAY BE CONTAINED IN PORTIONS OF THE ANTHOS CONFIG MANAGEMENT PRODUCT.

-----
" > "${out}"

# For each found license/notice, paste it into LICENSE.
while IFS='' read -r LINE || [[ -n "${LINE}" ]]; do
  package="$(echo "${LINE}" | sed -e "s/^vendor\///" -e "s/\/$licenseRegex$//")"
  {
    echo "The following software may be included in this product: $package. This software contains the following license and notice below:
"
    cat "${LINE}"
    echo "
-----
"
  } >> "${out}"
done < "${licenses}"

rm -r "${tmp}"
