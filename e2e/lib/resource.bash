

# Prints the resource version of a resource.
#
# Arguments
#   type: the resource type as specified to kubectl get
#   name: the resource name, for namespaced types use [namespace]:[name]
#
# Returns
#   Prints the current resource version for the resource.
function resource::resource_version() {
  local type=$1
  local name=$2

  args=("get" "${type}")
  if [[ "${name}" == *:* ]]; then
    ns="$(echo ${name} | sed -e 's/:.*//')"
    name="$(echo ${name} | sed -e 's/[^:]*://')"
    args+=("-n" "${ns}")
  fi
  args+=(${name})
  args+=("-ojsonpath='{.metadata.resourceVersion}'")

  kubectl "${args[@]}"
}

# Waits until a resource is updated. If timeout expires, this will be considered
# an error.
#
# Arguments
#   type: the resource type as specified to kubectl get
#   name: the resource name, for namespaced types use [namespace]:[name]
#   resource_version: the current resource version. Polling will continue until
#   this changes
#   timeout: (optional) number of seconds to wait before timing out.
#
function resource::wait_for_update() {
  local type=$1
  local name=$2
  local resource_version=$3
  local timeout=${4:-10}
  local ticks=$(( ${timeout} * 2 ))

  echo -n "Waiting for update of $type $name from version $resource_version"
  for i in $(seq 1 ${ticks}); do
    current=$(resource::resource_version $type $name)
    if [[ "$current" != "$resource_version" ]]; then
      echo
      return 0
    fi
    echo -n "."
    sleep 0.5
  done
  echo
  echo "Resource $type $name never updated from $resource_version"
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
  while [[ $# -gt 0 ]]; do
    local arg="${1:-}"
    shift
    case $arg in
      -n)
        namespace=${1:-}
        shift
      ;;
      -l)
        labels+=(${1:-})
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
    cmd+=(-n $namespace)
  fi

  echo "${cmd[@]}  ${#labels[@]}"
  output=$(${cmd[@]})
  [ "$status" -eq 0 ]
  local json="$output"
  if [[ "${#labels[@]}" != 0 ]]; then
    local label
    for label in "${labels[@]}"; do
      local key=$(echo "$label" | sed -e 's/=.*//')
      local value=$(echo "$label" | sed -e 's/[^=]*=//')
      output=$(echo "$json" | jq ".metadata.labels[\"${key}\"]")
      if [[ "$output" != "\"$value\"" ]]; then
        echo "Expected label $value, got $output"
        false
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

  local actual=$(resource::count "${args[@]}")
  if [[ $count != $actual ]]; then
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
    cmd+=(-n $namespace)
  fi
  if [[ "$selector" != "" ]]; then
    cmd+=(-l "$selector")
  fi

  local output="$("${cmd[@]}")"
  local count=0
  if [[ "$output" != "No resources found." ]]; then
    count=$(( $(echo "$output" | wc -l) - 1 ))
  fi
  echo $count
}

