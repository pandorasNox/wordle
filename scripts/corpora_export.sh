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

docker compose down -t 1

docker compose build

docker compose up -d

# ensure db connection
docker compose exec -T mariadb bash -c "MYSQL_HOST=mariadb MYSQL_USER=root MYSQL_PASSWORD=example MYSQL_PORT=3306 /scripts/check_db_con.sh"

#...
docker compose exec -T mariadb bash -c "/scripts/import.sh /datasets"
docker compose exec -T mariadb bash -c "/scripts/export.sh /datasets /tmp/dumps"

docker compose down -t 1
