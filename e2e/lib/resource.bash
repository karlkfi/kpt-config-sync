

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

