#!/bin/bash

echo "Running licenselinter: "
"${GOPATH}/bin/linux_amd64/licenselinter" -dir "$(pwd)"
echo "PASS"
echo
