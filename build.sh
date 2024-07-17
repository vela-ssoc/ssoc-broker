#!/usr/bin/env bash

CURRENT_DATE=$(date +'%Y%m%d')
BINARY_NAME=ssoc-broker-${CURRENT_DATE}$(go env GOEXE)

go build -o ${BINARY_NAME} -x -v -ldflags '-s -w' -trimpath ./main

if [ $? -eq 0 ]; then
    # 检查 upx 命令是否存在
    if command -v upx &> /dev/null; then
        upx -9 ${BINARY_NAME}
    fi
fi
