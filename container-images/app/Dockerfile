FROM node:20.12.1-alpine3.19 AS node

FROM golang:1.22-alpine AS builder

ENV WORKDIR /workdir
WORKDIR ${WORKDIR}

RUN apk add --no-cache \
    git \
    tmux \
    || apk update && apk upgrade

ENV AIR_VERSION 1.51.0
RUN go install github.com/cosmtrek/air@v${AIR_VERSION}

RUN mkdir -p /scripts
COPY ./container-images/app/tmux.sh /scripts/tmux.sh

# Copy Nodejs with dependencies from node image
COPY --from=node /usr/lib /usr/lib
COPY --from=node /usr/local/share /usr/local/share
COPY --from=node /usr/local/lib /usr/local/lib
COPY --from=node /usr/local/include /usr/local/include
COPY --from=node /usr/local/bin /usr/local/bin

COPY ./web/package*.json ${WORKDIR}/web/
RUN set -eu; cd ${WORKDIR}/web/; npm install; cd ${WORKDIR}

ENTRYPOINT ["/usr/bin/env", "ash"]
