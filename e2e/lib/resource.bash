#!/bin/bash

declare status

# Prints the resource version of a resource.
#
# Flags:
#   -n [namespace] the namespace to request
#
# Arguments
#   type: the resource type as specified to kubectl get
#   name: the resource name
#
# Returns
#   Prints the current resource version for the resource.
function resource::resource_version() {
  local args=()
  local cmd=(kubectl get)
  while [[ $# -gt 0 ]]; do
    local arg="${1:-}"
    shift
    case $arg in
      -n)
        cmd+=(-n "${1:-}")
        shift
      ;;
      *)
        args+=("$arg")
      ;;
    esac
  done

  local type="${args[0]}"
  local name="${args[1]}"
  [ -n "$type" ] || debug::error "must specify resource type (positional arg)"
  [ -n "$name" ] || debug::error "must specify resource name (positional arg)"
  (( ${#args[@]} == 2 )) || debug::error "resource::resource_version takes exactly two args"

  cmd+=("$type" "$name" "-ojsonpath='{.metadata.resourceVersion}'")
  "${cmd[@]}"
}

# Waits until a resource is updated. If timeout expires, this will be considered
# an error.
#
# Flags:
#   -n [namespace] the namespace of the resource
#   -t [timeout] the timeout for checking in seconds
#
# Arguments
#   type: the resource type as specified to kubectl get
#   name: the resource name, for namespaced types use [namespace]:[name]
#   resource_version: the current resource version. Polling will continue until
#   this changes
#   timeout: (optional) number of seconds to wait before timing out.
#
function resource::wait_for_update() {
  local args=()
  local cmd=(resource::resource_version)
  local timeout=10
  while [[ $# -gt 0 ]]; do
    local arg="${1:-}"
    shift
    case $arg in
      -n)
        cmd+=(-n "${1:-}")
        shift
      ;;
      -t)
        timeout=${1:-}
        shift
      ;;
      *)
        args+=("$arg")
      ;;
    esac
  done

  local type="${args[0]}"
  local name="${args[1]}"
  local resource_version="${args[2]}"
  [ -n "$type" ] || debug::error "must specify resource type (positional arg)"
  [ -n "$name" ] || debug::error "must specify resource name (positional arg)"
  [ -n "$resource_version" ] || debug::error "must specify resource version (positional arg)"
  (( ${#args[@]} == 3 )) || debug::error "resource::wait_for_update takes exactly three args"

  cmd+=("$type" "$name")

  local deadline
  (( deadline = $(date +%s) + timeout ))
  debug::log "Waiting for update of $type $name from version $resource_version"
  while (( "$(date +%s)" < "${deadline}" )); do
    local current
    current="$("${cmd[@]}")"
    if [[ "$current" != "$resource_version" ]]; then
      return 0
    fi
    sleep 0.1
  done
  debug::log "Resource $type $name never updated from $resource_version"
  return 1
}

# Checks the existence and labels of a given resource.
# Flags:
#  -n [namespace] get the count for a namespaced resource
#  -l [label] (repeated) specify a label that must exist on the resource
# Args:
#  [resource] the resource, eg clusterrole
#  [name] the resource name, eg, cluster-admin
# Example:
#  - Check that the namespace backend exists and has the label
#    "nomos.dev/marked=true"
#    resource::check ns backend -l "nomos.dev/marked=true"
#  - Check that the role pod-editor exists in the frontend namespace
#    resource::check -n frontend role pod-editor
function resource::check() {
  local arg
  local args=()
  local namespace=""
  local labels=()
  local annotations=()
  while [[ $# -gt 0 ]]; do
    local arg="${1:-}"
    shift
    case $arg in
      -n)
        namespace="${1:-}" # local
        shift
      ;;
      -l)
        labels+=("${1:-}")
        shift
      ;;
      -a)
        annotations+=("${1:-}")
        shift
      ;;
      *)
        args+=("$arg")
      ;;
    esac
  done

  local resource="${args[0]}"
  local name="${args[1]}"

  local cmd=("kubectl" "get" "$resource" "$name" -ojson)
  if [[ "$namespace" != "" ]]; then
    cmd+=(-n "$namespace")
  fi

  local output
  local status=0
  output=$("${cmd[@]}") || status=$?
  (( status == 0 )) || debug::error "Command" "${cmd[@]}" "failed, output ${output}"
  local json="$output"
  local key
  local value
  if [[ "${#labels[@]}" != 0 ]]; then
    local label
    for label in "${labels[@]}"; do
      key=$(sed -e 's/=.*//' <<< "$label")
      value=$(sed -e 's/[^=]*=//' <<< "$label")
      output=$(jq ".metadata.labels[\"${key}\"]" <<< "$json")
      if [[ "$output" != "\"$value\"" ]]; then
        debug::error "Expected label $value, got $output"
      fi
    done
  fi
  if [[ "${#annotations[@]}" != 0 ]]; then
    local annotation
    for annotation in "${annotations[@]}"; do
      key=$(sed -e 's/=.*//' <<< "$annotation")
      value=$(sed -e 's/[^=]*=//' <<< "$annotation")
      output=$(jq ".metadata.annotations[\"${key}\"]" <<< "$json")
      if [[ "$output" != "\"$value\"" ]]; then
        debug::error "Expected annotation $value, got $output"
      fi
    done
  fi
}

# Checks the count of a given resource
#
# Flags:
#  -n [namespace] get the count for a namespaced resource
#  -l [selector] use a selector during list
#  -r [resource] the resource, eg clusterrole
#  -c [count] the expected count, eg 5
function resource::check_count() {
  local args=()
  local count=""
  while [[ $# -gt 0 ]]; do
    local arg="${1:-}"
    shift
    case $arg in
      -c)
        count=${1:-}
        shift
      ;;
      *)
        args+=("$arg")
      ;;
    esac
  done
  [ -n "$count" ] || (echo "Must specify -c [count]"; return 1)

  local actual
  actual="$(resource::count "${args[@]}")"
  if (( count != actual )); then
    echo "Expected $count, got $actual"
    false
  fi
}

# Get the count of a given resource
#
# Flags:
#  -n [namespace] get the count for a namespaced resource
#  -l [selector] use a selector during list
#  -r [resource] the resource, eg clusterrole
function resource::count() {
  local namespace=""
  local selector=""
  local resource=""
  while [[ $# -gt 0 ]]; do
    local arg="${1:-}"
    shift
    case $arg in
      -n)
        namespace=${1:-}
        shift
      ;;
      -l)
        selector=${1:-}
        shift
      ;;
      -r)
        resource=${1:-}
        shift
      ;;
      *)
        echo "Unexpected arg $arg" >&3
        return 1
      ;;
    esac
  done
  [ -n "$resource" ] || (echo "Must specify -r [resource]" >&3; return 1)

  local cmd=("kubectl" "get" "$resource")
  if [[ "$namespace" != "" ]]; then
    cmd+=(-n "$namespace")
  fi
  if [[ "$selector" != "" ]]; then
    cmd+=(-l "$selector")
  fi

  local output
  local status=0
  output=$("${cmd[@]}") || status=$?
  (( status == 0 )) || debug::error "Command" "${cmd[@]}" "failed, output ${output}"
  local count=0
  if [[ "$output" != "No resources found." ]]; then
    count=$(( $(echo "$output" | wc -l) - 1 ))
  fi
  echo $count
}

