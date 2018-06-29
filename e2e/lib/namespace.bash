#!/bin/bash

# Helpers for creating namespaces/policyspaces in git.

# Directly creates a namespace on the cluster with optional label for nomos management.
#
# Arguments
#   name: The namespace name
# Parameters
#   -l [label] a label for the namespace (repeated)
#   -a [annotation] an annotation for the namespace (repeated)
function namespace::create() {
  local tmp="$(tempfile --directory ${BATS_TMPDIR} --prefix "ns-" --suffix ".yaml")"
  namespace::__genyaml -l 'nomos.dev/testdata=true' "$@" "${tmp}"
  echo "Creating Cluster Namespace:"
  cat ${tmp}
  echo
  kubectl apply -f ${tmp}
}

# Creates a policyspace directory in the git repo.
#
# Arguments
#   path: The path to the policyspace directory under acme with the last portion being the policyspace name.
#
function namespace::declare_policyspace() {
  local path=$1
  local dst=acme/${path}/OWNERS
  local abs_dst=${TEST_REPO}/${dst}
  mkdir -p $(dirname ${abs_dst})
  touch ${abs_dst}
  git -C ${TEST_REPO} add ${dst}
}

# Creates a namespace directory and yaml in the git repo.  If policyspaces are
# required based on the path they will be implicitly created as well.
#
# Arguments
#   path: The path to the namesapce directory under acme with the last portion being the namesapce name.
# Parameters
#   -l [label] a label for the namespace (repeated)
#   -a [annotation] an annotation for the namespace (repeated)
function namespace::declare() {
  local args=()
  local genyaml_args=()
  while [[ $# -gt 0 ]]; do
    local arg="${1:-}"
    shift
    case $arg in
      -a)
        genyaml_args+=("-a" "${1:-}")
        shift
      ;;
      -l)
        genyaml_args+=("-l" "${1:-}")
        shift
      ;;
      *)
        args+=("${arg}")
      ;;
    esac
  done

  local path=${args[0]:-}
  [ -n "$path" ] || debug::error "Must specify namespace name"

  local name=$(basename ${path})
  local dst=acme/${path}/namespace.yaml
  local abs_dst=${TEST_REPO}/${dst}
  genyaml_args+=("${name}" "${abs_dst}")
  namespace::__genyaml "${genyaml_args[@]}"
  echo "Declaring namespace:"
  cat "${abs_dst}"
  git -C ${TEST_REPO} add ${dst}
}

# Designates a reserved namespace in the git repo in the
# nomos-reserved-namespaces.yaml.  This will overwrite any previous designations
# with only this designation.
#
# Arguments
#   name: The name of the reserved namespace.
#
function namespace::declare_reserved() {
  local name=$1
  local dst=acme/nomos-reserved-namespaces.yaml

  cat > ${TEST_REPO}/${dst} <<- EOM
apiVersion: v1
kind: ConfigMap
metadata:
  name: nomos-reserved-namespaces
data:
  ${name}: reserved
EOM
  git -C ${TEST_REPO} add ${dst}
}

# Checks that a namespace exists on the cluster.
#
# Arguments
#   name: The name of the namespace
# Params
#   -l [label]: the value of a label to check, formatted as "key=value"
#   -a [annotation]: an annotation to check for, formatted as "key=value"
function namespace::check_exists() {
  local args=()
  local check_args=()
  while [[ $# -gt 0 ]]; do
    local arg="${1:-}"
    shift
    case $arg in
      -a)
        check_args+=(-a "${1:-}")
        shift
      ;;
      -l)
        check_args+=(-l "${1:-}")
        shift
      ;;
      *)
        args+=("${arg}")
      ;;
    esac
  done

  local name=${args[0]:-}
  [ -n "$name" ] || debug::error "Must specify namespace name"

  wait::for_success "kubectl get ns ${name}"

  if [[ "${#check_args[@]}" != 0 ]]; then
    resource::check ns ${name} "${check_args[@]}"
  fi
}

# Checks that a namespace does not exist
#
# Arguments
#   name: The name of the namespace
function namespace::check_not_found() {
  local ns=$1
  wait::for_failure "kubectl get ns ${ns}"
  run kubectl get ns ${ns}
  assert::contains "NotFound"
}

# Checks that a namespace has a warning event.
#
# Arguments
#   name: The name of the namespace
function namespace::check_warning() {
  local ns=$1
  echo "WARN: checking for namespace warning not yet implemented" 1>&2
}

# Checks that a namespace has no warning event.
#
# Arguments
#   name: The name of the namespace
function namespace::check_no_warning() {
  local ns=$1
  echo "WARN: checking for no namespace warning not yet implemented" 1>&2
}

# Internal function
# Generates namespace yaml
# Parameters
#   -l [label] a label for the namespace (repeated)
#   -a [annotation] an annotation for the namespace (repeated)
# Arguments
#   [name] the namespace name
#   [filename] the file to create
function namespace::__genyaml() {
  debug::log "namespace::__genyaml $@"
  local args=()
  local annotations=()
  local labels=()
  while [[ $# -gt 0 ]]; do
    local arg="${1:-}"
    shift
    case $arg in
      -a)
        annotations+=("${1:-}")
        shift
      ;;
      -l)
        labels+=("${1:-}")
        shift
      ;;
      *)
        args+=("${arg}")
      ;;
    esac
  done
  local name="${args[0]:-}"
  local filename="${args[1]:-}"
  [ -n "$name" ] || debug::error "Must specify namespace name"
  [ -n "$filename" ] || debug::error "Must specify yaml filename"

  mkdir -p "$(dirname "$filename")"
  cat > $filename <<- EOM
apiVersion: v1
kind: Namespace
metadata:
  name: ${name}
EOM
  if [[ "${#labels[@]}" != 0 ]]; then
    echo "  labels:" >> $filename
    local label
    for label in "${labels[@]}"; do
      namespace::__yaml::kv "    " "$label" >> $filename
    done
  fi
  if [[ "${#annotations[@]}" != 0 ]]; then
    echo "  annotations:" >> $filename
    local annotation
    for annotation in "${annotations[@]}"; do
      namespace::__yaml::kv "    " "$annotation" >> $filename
    done
  fi
}

# Internal. Creates a yaml key-value pair for labels / annotations
# Params
#   [indent] spaces for indent
#   [value] keyvalue formatted as "key=value"
function namespace::__yaml::kv() {
  local indent=${1:-}
  local keyval="${2:-}"
  local key=$(echo "${keyval}" | sed -e 's/=.*//')
  local value=$(echo "${keyval}" | sed -e 's/[^=]*=//')
  case "$value" in
    [0-9]*|true|false)
      value="\"$value\""
      ;;
  esac
  echo "    $key: $value"
}

