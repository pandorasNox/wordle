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

# drop db if exist (ensure clean plate)
docker compose exec -T mariadb bash -c "mariadb -uroot -p'example' -e 'DROP DATABASE IF EXISTS eng_news_2023_10K'"

# import corpora data
docker compose exec -T mariadb bash -c "cd /datasets/eng_news_2023_10K; mariadb -uroot -p'example' < eng_news_2023_10K-import.sql"

# export start

tmpDir=$(mktemp -d)
queryFileName=query.txt
tmpQueryFilePath=${tmpDir}/${queryFileName}

exportFilePath=/tmp/dumps/corpora-eng_news_2023_10K-export.txt

# ensure export file doesn't exist (can not overwrite via query...)
docker compose exec -T mariadb bash -c "rm ${exportFilePath} 2> /dev/null || true"

cat << EOF > ${tmpQueryFilePath}
SELECT word FROM words
WHERE CHAR_LENGTH(word) = 5
  AND word RLIKE "^[a-z]+$"
  AND freq > 1
INTO OUTFILE '${exportFilePath}'
  LINES TERMINATED BY '\n'
EOF

# copy query file into db container
docker compose cp ${tmpQueryFilePath} mariadb:/tmp/${queryFileName}

# cleanup tmp files
rm ${tmpQueryFilePath}
rmdir ${tmpDir}

# run export
docker compose exec -T mariadb bash -c \
  "cat /tmp/${queryFileName} | mariadb -uroot -p'example' eng_news_2023_10K"
