#!/usr/bin/env bash

set -e

BASE_NAME=$(basename $(pwd))
VERSION=$(TZ=UTC git log -1 --format="%cd" --date=format-local:"v%-y.%-m.%-d")
SHORT_SHA=$(git rev-parse --short HEAD)
BINARY_NAME="${BASE_NAME}_$(go env GOOS)-$(go env GOARCH)_${VERSION}+${SHORT_SHA}$(go env GOEXE)"
CURRENT_TIME=$(date --rfc-email)

LD_FLAGS="-s -w -X 'github.com/vela-ssoc/ssoc-common/banner.compileTime=${CURRENT_TIME}'"
CGO_ENABLED=1 go build -o "${BINARY_NAME}" -trimpath -ldflags "${LD_FLAGS}" ./main

echo "编译完成：${BINARY_NAME}"