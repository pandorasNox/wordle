#!/usr/bin/env bash

set -o errexit
set -o nounset
# set -o xtrace

if set +o | grep -F 'set +o pipefail' > /dev/null; then
  # shellcheck disable=SC3040
  set -o pipefail
fi

if set +o | grep -F 'set +o posix' > /dev/null; then
  # shellcheck disable=SC3040
  set -o posix
fi

# -----------------------------------------------------------------------------

APP_PORT=9026

# -----------------------------------------------------------------------------

#   up                ...
#   down              ...
__usage="
Usage: $(basename $0) [OPTIONS]

Options:
  --help|-h         show help
  watch             start go server & reload server upon file chnages
"

# -----------------------------------------------------------------------------

func_watch() {
  docker run -it --rm \
    -w "/workdir" -v "${PWD}":"/workdir" \
    -p "${APP_PORT}":"${APP_PORT}" \
    -e PORT="${APP_PORT}" \
    docker.io/cosmtrek/air "$@"
}

# -----------------------------------------------------------------------------

if [ -z "$*" ]
then
  echo "$__usage"
else
    if [ $1 == "--help" ] || [ $1 == "-h" ]
    then
        echo "$__usage"
    fi

    if [ $1 == "watch" ]
    then
      func_watch
    #   func_watch -c "--build.exclude_dir 'bin'" 
    fi
fi
