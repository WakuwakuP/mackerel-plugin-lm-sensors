#!/bin/sh

set -eu

GOLANG_VERSION=1.25.1
GOXZ_VERSION=v0.10.1
NAME=mackerel-plugin-lm-sensors
RELEASE_VERSION=${1:-dev}
PROJECT_DIR=$(pwd)
DOCKER_WORKDIR="/go/$NAME"

case "$(uname -s)" in
  MINGW* | MSYS* | CYGWIN*)
    PROJECT_DIR=$(pwd -W)
    export MSYS_NO_PATHCONV=1
    ;;
esac

docker run --rm \
  -e "RELEASE_VERSION=$RELEASE_VERSION" \
  -e "GOXZ_VERSION=$GOXZ_VERSION" \
  -v "$PROJECT_DIR:$DOCKER_WORKDIR" \
  -w "$DOCKER_WORKDIR" \
  "golang:$GOLANG_VERSION" \
  sh -c '
    set -eu
    go install "github.com/Songmu/goxz/cmd/goxz@$GOXZ_VERSION"
    git config --global --add safe.directory "$(pwd)"
    goxz -d "dist/$RELEASE_VERSION" -n mackerel-plugin-lm-sensors -z -os linux -arch amd64,arm64 .
  '
