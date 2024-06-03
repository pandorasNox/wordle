FROM mariadb:11.3.2

RUN set -e; \
    apt-get update; \
    apt-get install -y \
        curl \
    ;

COPY ./corpora_download.sh /scripts/corpora_download.sh
RUN set -e; mkdir -p /datasets; /scripts/corpora_download.sh /datasets

RUN mkdir -p /scripts
COPY ./check_db_con.sh /scripts/check_db_con.sh
