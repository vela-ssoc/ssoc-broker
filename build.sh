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

# macOS 下未测试
NOW=$(date --iso-8601=seconds)
OSNAME=$(uname -s)
if [ $OSNAME == Darwin ]; then
    NOW=$(date)
fi

LDFLAGS="-s -w -extldflags -static -X 'github.com/vela-ssoc/vela-broker/banner.compileTime=$NOW'"
go build -o ${BIN_NAME} -trimpath -v -ldflags "$LDFLAGS" ./main

# 检查上一步是否执行成功
if [[ $? -eq 0 ]]; then
    # 检查 upx 命令是否存在
    if command -v upx &> /dev/null; then
        upx -9 ${BIN_NAME}
    fi
fi

echo "编译打包结束"
