#!/usr/bin/env bash

#if [ ! -f /usr/local/lib/libpcap.a ]; then
#    echo "libpcap.a 不存在，开始编译安装 libpcap"
#    yum install -y flex bison
#    wget https://www.tcpdump.org/release/libpcap-1.10.5.tar.gz
#    tar xf libpcap-1.10.5.tar.gz
#    cd libpcap-1.10.5
#    ./configure --disable-shared
#    make
#    make install
#
#    cd ..
#    rm -rf libpcap-1.10.5
#fi
#
#
#export CGO_LDFLAGS="-L/usr/local/lib"
#export CGO_CFLAGS="-I/usr/local/include"

BASE_NAME=$(basename $(pwd))
VERSION=$(TZ=UTC git log -1 --format="%cd" --date=format-local:"v%-y.%-m.%-d-%H%M%S")
CURRENT_TIME=$(date -u +"%FT%TZ")

BINARY_NAME="${BASE_NAME}_$(go env GOOS)-$(go env GOARCH)_${VERSION}$(go env GOEXE)"
LD_FLAGS="-s -w -linkmode external -extldflags '-static'-X 'github.com/vela-ssoc/ssoc-common/banner.compileTime=${CURRENT_TIME}'"
CGO_ENABLED=1 go build -o "${BINARY_NAME}" -tags osusergo,netgo -trimpath -ldflags "${LD_FLAGS}" ./main

echo "编译完成：${BINARY_NAME}"
