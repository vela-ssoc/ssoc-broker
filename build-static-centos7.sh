#!/usr/bin/env bash

(
yum install -y flex bison
wget https://www.tcpdump.org/release/libpcap-1.10.5.tar.gz
tar xf libpcap-1.10.5.tar.gz
cd libpcap-1.10.5
./configure --disable-shared
make
make install
)

# 1. 获取程序名。
DIR_NAME=$(basename $(pwd))
VER=$(TZ="Europe/London" date -d "$(git log -1 --format=%cd --date=iso)"  +"%y.%-m.%-d-%H%M%S")
BIN_NAME=${DIR_NAME}"-"v$VER$(go env GOEXE)
echo "程序名为："${BIN_NAME}

export CGO_ENABLED=1
export CGO_LDFLAGS="-L/usr/local/lib"
export CGO_CFLAGS="-I/usr/local/include"

NOW=$(date)
LDFLAGS="-linkmode external -extldflags '-static' -s -w -X 'github.com/vela-ssoc/ssoc-broker/banner.compileTime=$NOW'"
go build -o ${BIN_NAME} -trimpath -v -ldflags "$LDFLAGS" ./main

echo "编译结束"


