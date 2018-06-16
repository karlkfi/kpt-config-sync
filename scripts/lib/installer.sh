#!/bin/bash
# BEGIN: Inline installer library.

# installer::bool converts the 'on' and 'off' permissions from argbash into
# the 'true' and 'false' respectively.
function installer::bool() {
	case $1 in
	off) echo "false"
			;;
	on) echo "true"
			;;
	default)
		echo "### Can not be parsed as on/off value: ${@}"
		exit 200
	esac
}

# installer::optf optionally formats $@ into $1 if $2 is nonempty.  Otherwise
# returns nothing.
function installer::optf() {
	if [[ "$2" != "" ]]; then
			printf -- "$@"
	fi
}

# installer::abspath converts a filename in $1 to an absolute path.
function installer::abspath() {
  case "$1" in
    /*) printf '%s\n' "$1";;
    *) printf '%s\n' "$PWD/$1";;
  esac
}
# END: Inline installer library.

