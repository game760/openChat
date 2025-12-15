#!/bin/bash

# 创建输出目录
mkdir -p bin

# 设置编译参数
GOOS=windows
GOARCH=amd64
OUTPUT="bin/easychat_windows_amd64.exe"

# 开始编译
echo "开始编译 windows 版本..."
GOOS=$GOOS GOARCH=$GOARCH go build -ldflags "-s -w" -o $OUTPUT ./main.go

# 检查编译结果
if [ $? -eq 0 ]; then
    echo "编译成功！输出文件：$OUTPUT"
else
    echo "编译失败！"
    exit 1
fi