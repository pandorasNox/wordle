FROM golang:1.22-alpine AS builder

RUN apk add --no-cache \
    git \
    tmux \
    || apk update && apk upgrade

ENV AIR_VERSION 1.51.0
RUN go install github.com/cosmtrek/air@v${AIR_VERSION}

RUN mkdir -p /scripts
COPY ./tmux.sh /scripts/tmux.sh

ENTRYPOINT ["/usr/bin/env", "ash"]
