FROM mariadb:11.3.2

RUN set -e; \
    apt-get update; \
    apt-get install -y \
        curl \
    ;

RUN set -e; \
    curl \
        --output '/tmp/eng_news_2023_10K.tar.gz' \
        https://downloads.wortschatz-leipzig.de/corpora/eng_news_2023_10K.tar.gz \
    ;
RUN set -e; \
    tar -xvzf /tmp/eng_news_2023_10K.tar.gz -C /tmp/;

RUN mkdir -p /scripts
COPY ./check_db_con.sh /scripts/check_db_con.sh
