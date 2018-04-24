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

NOMOS_ROOT=$(dirname "${BASH_SOURCE}")/..

# The tool doesn't gracefully handle multiple GOPATH values, so this will get
# the first and last values from GOPATH.
GOBASE=$(echo $GOPATH | sed 's/:.*//')
GOWORK=$(echo $GOPATH | sed 's/.*://')
REPO="github.com/google/nomos"

# Comma separted list of APIs to generate for clientset.
INPUT_BASE="${REPO}/pkg/api"
INPUT_APIS="policyhierarchy/v1"

# Nomos proto dependencies.
K8S_APIS_PROTO=(
  k8s.io/apimachinery/pkg/util/intstr
  +k8s.io/apimachinery/pkg/api/resource
  +k8s.io/apimachinery/pkg/runtime/schema
  +k8s.io/apimachinery/pkg/runtime
  k8s.io/apimachinery/pkg/apis/meta/v1
  k8s.io/apimachinery/pkg/apis/meta/v1alpha1
  k8s.io/api/core/v1
  k8s.io/api/rbac/v1
  k8s.io/api/extensions/v1beta1
)

# Where to put the generated client set
OUTPUT_BASE="${GOWORK}/src"
OUTPUT_CLIENT="${REPO}/clientgen"
CLIENTSET_NAME=policyhierarchy

BOILERPLATE="$(dirname ${0})/boilerplate.go.txt"

LOGGING_FLAGS=${LOGGING_FLAGS:- --logtostderr -v 5}
if ${SILENT:-false}; then
  LOGGING_FLAGS=""
fi

tools=()
for tool in client-gen deepcopy-gen informer-gen lister-gen go-to-protobuf go-to-protobuf/protoc-gen-gogo; do
  tools+=("k8s.io/code-generator/cmd/${tool}")
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

if [[ -z "$(which protoc)" || "$(protoc --version)" != "libprotoc 3."* ]]; then
  echo "ERROR:"
  echo "Generating protobuf requires protoc 3.0.0-beta1 or newer. Please download and"
  echo "install the platform appropriate Protobuf package for your OS: "
  echo
  echo "  https://github.com/google/protobuf/releases"
  echo
  echo "WARNING: Protobuf changes are not being validated"
  exit 1
fi

echo "Using GOPATH base ${GOBASE}"
echo "Using GOPATH work ${GOWORK}"

${GOBASE}/bin/client-gen \
  --input-base "${INPUT_BASE}" \
  --input="${INPUT_APIS}" \
  --clientset-name="${CLIENTSET_NAME}" \
  --output-base="${OUTPUT_BASE}" \
  --go-header-file="${BOILERPLATE}" \
  --clientset-path "${OUTPUT_CLIENT}"

echo "Generating installer deepcopy"

${GOBASE}/bin/deepcopy-gen \
  ${LOGGING_FLAGS} \
  --input-dirs="${REPO}/pkg/installer/config" \
  --output-file-base="types.generated" \
  --go-header-file="${BOILERPLATE}" \
  --output-base="${OUTPUT_BASE}"

echo "Generating APIs"

for api in $(echo "${INPUT_APIS}" | tr ',' ' '); do
  echo "Generating API: ${api}"
  echo "deepcopy"
  # Creates types.generated.go
  ${GOBASE}/bin/deepcopy-gen \
    ${LOGGING_FLAGS} \
    --input-dirs="${INPUT_BASE}/${api}" \
    --output-file-base="types.generated" \
    --go-header-file="${BOILERPLATE}" \
    --output-base="${OUTPUT_BASE}"

  echo "lister"
  ${GOBASE}/bin/lister-gen \
    ${LOGGING_FLAGS} \
    --input-dirs="${INPUT_BASE}/${api}" \
    --output-base="$GOWORK/src" \
    --go-header-file="${BOILERPLATE}" \
    --output-package="${OUTPUT_CLIENT}/listers"

  echo "informer"
  ${GOBASE}/bin/informer-gen \
    ${LOGGING_FLAGS} \
    --input-dirs="${INPUT_BASE}/${api}" \
    --versioned-clientset-package="${OUTPUT_CLIENT}/${CLIENTSET_NAME}" \
    --listers-package="${OUTPUT_CLIENT}/listers" \
    --output-base="$GOWORK/src" \
    --go-header-file="${BOILERPLATE}" \
    --output-package="${OUTPUT_CLIENT}/informers"

  echo "protobuf"
  ${GOBASE}/bin/go-to-protobuf \
    ${LOGGING_FLAGS} \
    --proto-import="${NOMOS_ROOT}/vendor" \
    --proto-import="${NOMOS_ROOT}/third_party/protobuf" \
    --packages="+${INPUT_BASE}/${api}" \
    --apimachinery-packages=$(IFS=, ; echo "${K8S_APIS_PROTO[*]}") \
    --output-base="$GOWORK/src" \
    --go-header-file="${BOILERPLATE}"
done

# go-to-protobuf changes generated proto given in K8S_APIS_PROTO
# Revert these unneeded changes.
find ${NOMOS_ROOT}/vendor \( -name "generated.proto" -o -name "generated.pb.go" \) \
    -exec git checkout {} \;

echo "Generation Completed!"
