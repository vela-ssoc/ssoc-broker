#!/usr/bin/env bash

VERSION=$(TZ=UTC git log -1 --format="%cd" --date=format-local:"v%-y.%-m.%-d-t%H%M%S")

OS_TYPE=$(uname -s)
case "$OS_TYPE" in
  MINGW* | MSYS* | CYGWIN* | Windows_NT) # Windows 不支持 %-y 格式。
    VERSION=$(TZ=UTC git log -1 --format="%cd" --date=format-local:"v%y.%m.%d-t%H%M%S")
  ;;
esac

BASE_NAME=$(basename $(pwd))
CURRENT_TIME=$(date -u +"%FT%TZ")

BINARY_NAME="${BASE_NAME}_$(go env GOOS)-$(go env GOARCH)_${VERSION}$(go env GOEXE)"
LD_FLAGS="-s -w -extldflags=-static -X 'github.com/vela-ssoc/ssoc-common/banner.compileTime=${CURRENT_TIME}'"
go build -o "${BINARY_NAME}" -trimpath -v -ldflags "${LD_FLAGS}" ./main
