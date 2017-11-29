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
REPO="github.com/google/stolos"

# Comma separted list of APIs to generate for clientset.
INPUT_BASE="${REPO}/pkg/api"
INPUT_APIS="policyhierarchy/v1"

# Where to put the generated client set
OUTPUT_BASE="${GOWORK}/src"
OUTPUT_CLIENT="${REPO}/pkg/client"
CLIENTSET_NAME=policyhierarchy

BOILERPLATE="$(dirname ${0})/boilerplate.go.txt"

LOGGING_FLAGS=${LOGGING_FLAGS:- --logtostderr -v 5}
if ${SILENT:-false}; then
  LOGGING_FLAGS=""
fi

tools=()
for tool in client deepcopy informer lister; do
  tools+=("k8s.io/code-generator/cmd/${tool}-gen")
done

branch=release-1.9
echo "Checking out codegen branch ${branch}"
checkout="git -C ${GOBASE}/src/k8s.io/code-generator \
    checkout -B ${branch} origin/${branch}"
if ! ${checkout} &> /dev/null; then
  echo "Fetching gen tools and checking out appropriate branch..."
  go get -d -u "${tools[@]}"
  ${checkout}
fi

echo "Building gen tools..."
go install "${tools[@]}"

echo "Using GOPATH base ${GOBASE}"
echo "Using GOPATH work ${GOWORK}"

${GOBASE}/bin/client-gen \
  --input-base "${INPUT_BASE}" \
  --input="${INPUT_APIS}" \
  --clientset-name="${CLIENTSET_NAME}" \
  --output-base="${OUTPUT_BASE}" \
  --go-header-file="${BOILERPLATE}" \
  --clientset-path "${OUTPUT_CLIENT}"

for api in $(echo "${INPUT_APIS}" | tr ',' ' '); do
  # Creates types.generated.go
  ${GOBASE}/bin/deepcopy-gen \
    ${LOGGING_FLAGS} \
    --input-dirs="${INPUT_BASE}/${api}" \
    --output-file-base="types.generated" \
    --go-header-file="${BOILERPLATE}" \
    --output-base="${OUTPUT_BASE}"

  ${GOBASE}/bin/lister-gen \
    ${LOGGING_FLAGS} \
    --input-dirs="${INPUT_BASE}/${api}" \
    --output-base="$GOWORK/src" \
    --go-header-file="${BOILERPLATE}" \
    --output-package="${OUTPUT_CLIENT}/listers"

  ${GOBASE}/bin/informer-gen \
    ${LOGGING_FLAGS} \
    --input-dirs="${INPUT_BASE}/${api}" \
    --versioned-clientset-package="${OUTPUT_CLIENT}/${CLIENTSET_NAME}" \
    --listers-package="${OUTPUT_CLIENT}/listers" \
    --output-base="$GOWORK/src" \
    --go-header-file="${BOILERPLATE}" \
    --output-package="${OUTPUT_CLIENT}/informers"
done
