#!/bin/sh

set -eu

GOLANG_VERSION=1.23.1
NAME=mackerel-plugin-lm-sensors

docker run --rm \
  -v "$(pwd):/go/$NAME" \
  -w "/go/$NAME" \
  "golang:$GOLANG_VERSION" \
  sh -c '
    set -eu
    go install github.com/Songmu/goxz/cmd/goxz@latest
    git config --global --add safe.directory "$(pwd)"
    goxz -z -os linux -arch amd64,arm64
  '
