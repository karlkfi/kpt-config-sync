#!/bin/bash

# Fail on any error.
set -e
# Display commands being run.
set -x

# Kokoro checks out the code into a directory called "stolos"
go test stolos/...
