

function debug::log() {
  local src=$(basename ${BASH_SOURCE[1]})
  local line=${BASH_LINENO[0]}
  local func="<global>"
  if [[ ${#FUNCNAME[@]} > 1 ]]; then
    func="(${FUNCNAME[1]})"
  fi
  echo "# $src:$line $func $@" >&3
}
