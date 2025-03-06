#!/bin/bash

# 1. 获取程序名。
DIR_NAME=$(basename $(pwd))
BIN_NAME=${DIR_NAME}"-"$(date +%Y%m%d)$(go env GOEXE)
echo "程序名为："${BIN_NAME}

# 2. 如果执行的是清理命令，清理完就退出。
go clean -cache
if [ "$1" = "clean" ]; then
    rm -rf ${DIR_NAME}*
    echo "已清理"
    exit 0
fi

NOW=$(date)
LDFLAGS="-s -w -X 'github.com/vela-ssoc/vela-broker/banner.compileTime=$NOW'"
go build -o ${BIN_NAME} -trimpath -v -ldflags "$LDFLAGS" ./main

echo "编译打包结束"
