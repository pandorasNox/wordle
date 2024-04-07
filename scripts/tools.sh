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
CLI_CONTAINER_NAME=wordle_cli_con

func_cli() {
  CONTAINER_NAME=${CLI_CONTAINER_NAME}

  if ! (docker ps --format "{{.Names}}" | grep "${CONTAINER_NAME}"); then
    func_build_devtools_img "${DEVTOOLS_IMG_NAME}"

    func_start "${DEVTOOLS_IMG_NAME}" "${CONTAINER_NAME}"
  fi

  docker exec -it ${CONTAINER_NAME} ash
}

func_build_devtools_img() {
  IMG_NAME=${1:?"first param missing, which is expected to be a chosen image name"}

  docker build \
    -t "${IMG_NAME}" \
    -f container-images/dev-tools/Dockerfile \
    .

  printf '%s' "${IMG_NAME}"
}

func_start() {
  IMG_NAME=${1:?"first param missing, which is expected to be a chosen image name"}
  CONTAINER_NAME=${2:?"second param missing, which is expected to be a chosen container name"}

  if ! (docker ps --format "{{.Names}}" | grep "${CONTAINER_NAME}"); then
    docker run -d --rm \
      --name ${CONTAINER_NAME} \
      -w "/workdir" -v "${PWD}":"/workdir" \
      --entrypoint=ash \
      ${IMG_NAME} -c "while true; do sleep 2000000; done"
      # -v "${PWD}/tmp/local_go_dev_dir":"/go" \
  fi
}

func_watch() {
  func_build_devtools_img "${DEVTOOLS_IMG_NAME}"

  docker run -it --rm \
    -w "/workdir" -v "${PWD}":"/workdir" \
    -p "${APP_PORT}":"${APP_PORT}" \
    -e PORT="${APP_PORT}" \
    --entrypoint=ash \
    "${DEVTOOLS_IMG_NAME}" -c "air --build.cmd 'npx tsc --project web/app/tsconfig.json && go build -buildvcs=false -o ./tmp/main' --build.bin './tmp/main' -build.include_ext 'go,tpl,tmpl,templ,html,ts,json' -build.exclude_dir 'assets,tmp,vendor,node_modules'"
}

func_test() {
  CONTAINER_NAME=${CLI_CONTAINER_NAME}

  if ! (docker ps --format "{{.Names}}" | grep "${CLI_CONTAINER_NAME}"); then
    func_build_devtools_img "${DEVTOOLS_IMG_NAME}"

    func_start "${DEVTOOLS_IMG_NAME}" "${CONTAINER_NAME}"
  fi

  docker exec -t ${CONTAINER_NAME} ash -c "$@"
}

func_down() {
  docker stop -t1 "${CLI_CONTAINER_NAME}"
}

func_skopeo_cli() {
  docker run -it --rm --entrypoint=bash quay.io/skopeo/stable:v1.14.2
}

func_typescript_build() {
  CONTAINER_NAME=${CLI_CONTAINER_NAME}

  if ! (docker ps --format "{{.Names}}" | grep "${CLI_CONTAINER_NAME}"); then
    func_build_devtools_img "${DEVTOOLS_IMG_NAME}"

    func_start "${DEVTOOLS_IMG_NAME}" "${CONTAINER_NAME}"
  fi

  docker exec -t ${CONTAINER_NAME} ash -ce "npm install; npx tsc --project web/app/tsconfig.json;"
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

    if [ $1 == "cli" ]
    then
      func_cli
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

    if [ $1 == "img" ]
    then
      func_build_devtools_img "${DEVTOOLS_IMG_NAME}"
    fi

    if [ $1 == "tsc" ]
    then
      func_typescript_build
    fi
fi
