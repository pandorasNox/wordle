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
  test              start go docker container + run go tests
  down              stop + delete all started local docker container
"

# -----------------------------------------------------------------------------

DEVTOOLS_IMG_NAME=wordle_dev_tools

func_watch() {
  func_build_devtools_img "${DEVTOOLS_IMG_NAME}"

  docker run -it --rm \
    -w "/workdir" -v "${PWD}":"/workdir" \
    -p "${APP_PORT}":"${APP_PORT}" \
    -e PORT="${APP_PORT}" \
    --entrypoint=ash \
    "${DEVTOOLS_IMG_NAME}" -c "air --build.cmd 'go build -buildvcs=false -o ./tmp/main' --build.bin './tmp/main'"
}

func_build_devtools_img() {
  IMG_NAME=${1:?"first param missing, which is expected to be a chosen image name"}

  docker build \
    -t "${IMG_NAME}" \
    -f container-images/dev-tools/Dockerfile \
    container-images/dev-tools

  printf '%s' "${IMG_NAME}"
}

func_test() {
  if ! (docker ps --format "{{.Names}}" | grep "wordle_test_con"); then
    docker run -d --rm \
      --name wordle_test_con \
      -w "/workdir" -v "${PWD}":"/workdir" \
      -v "${PWD}/tmp/local_go_dev_dir":"/go" \
      --entrypoint=bash \
      docker.io/cosmtrek/air -c "while true; do sleep 2000000; done"
  fi

  docker exec -t wordle_test_con bash -c "$@"
}

func_down() {
  docker stop -t1 wordle_test_con
}

func_skopeo_cli() {
  docker run -it --rm --entrypoint=bash quay.io/skopeo/stable:v1.14.2
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
    fi

    if [ $1 == "test" ]
    then
      func_test "go test ."
    fi

    if [ $1 == "down" ]
    then
      func_down
    fi

    if [ $1 == "skocli" ]
    then
      func_skopeo_cli
    fi
fi
