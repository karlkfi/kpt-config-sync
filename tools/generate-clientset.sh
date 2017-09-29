#!/bin/bash
#
# This command will generate the clientset for our custom resources.
#
# For clientset
# https://github.com/kubernetes/community/blob/master/contributors/devel/generating-clientset.md
#
# For deepcopy, there are no docs.
#
# If you want to generate a new CRD, create doc.go, types.go, register.go and
# use the existing files for a reference on all the // +[comment] things you
# need to add so it will work properly, then add it to INPUT_APIS
#
# Note that there are a bunch of other generators in
# k8s.io/code-generator/tree/master/cmd that we might have to use in the future.
#

set -euo pipefail

# The tool doesn't gracefully handle multiple GOPATH values, so this will get
# the first and last values from GOPATH.
GOBASE=$(echo $GOPATH | sed 's/:.*//')
GOWORK=$(echo $GOPATH | sed 's/.*://')

# Comma separted list of APIs to generate for clientset.
INPUT_BASE="github.com/google/stolos/pkg/api"
INPUT_APIS="policyhierarchy/v1"

# Where to put the generated client set
OUTPUT_BASE="${GOWORK}/src"
OUTPUT_CLIENT="github.com/google/stolos/pkg/client"
CLIENTSET_NAME=policyhierarchy

echo "Building gen tools..."
go install k8s.io/code-generator/cmd/client-gen
go install k8s.io/code-generator/cmd/deepcopy-gen
#go install k8s.io/code-generator/cmd/informer-gen
#go install k8s.io/code-generator/cmd/lister-gen

echo "Using GOPATH base ${GOBASE}"
echo "Using GOPATH work ${GOWORK}"

${GOBASE}/bin/client-gen \
  --input-base "${INPUT_BASE}" \
  --input="${INPUT_APIS}" \
  --clientset-name="${CLIENTSET_NAME}" \
  --output-base="${OUTPUT_BASE}" \
  --clientset-path "${OUTPUT_CLIENT}"


for api in $(echo "${INPUT_APIS}" | tr ',' ' '); do
  # Creates types.generated.go
  ${GOBASE}/bin/deepcopy-gen \
    --logtostderr \
    -v 5 \
    --input-dirs="${INPUT_BASE}/${api}" \
    --output-file-base="types.generated" \
    --output-base="${OUTPUT_BASE}"

  #${GOBASE}/bin/lister-gen \
  #  --logtostderr \
  #  -v 5 \
  #  --input-dirs="${INPUT_BASE}/${api}" \
  #  --output-base="$GOWORK/src" \
  #  --output-package="${OUTPUT_CLIENT}/listers"

  #${GOBASE}/bin/informer-gen \
  #  --logtostderr \
  #  -v 5 \
  #  --input-dirs="${INPUT_BASE}/${api}" \
  #  --versioned-clientset-package="${OUTPUT_CLIENT}/${CLIENTSET_NAME}" \
  #  --listers-package="${OUTPUT_CLIENT}/listers" \
  #  --output-base="$GOWORK/src" \
  #  --output-package="${OUTPUT_CLIENT}/informers"
done
