#!/bin/bash
# client_build.sh - 跨平台编译脚本
# 编译目标: OpenWrt (x86_64), Windows (AMD64), Android (ARM64)

set -e

# 获取操作系统类型
OS="$(uname -s)"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
OUTPUT_DIR="$SCRIPT_DIR/smart_spend/dist"
CLIENT_DIR="$SCRIPT_DIR/client"

case "$OS" in
    Linux*)     OS_TYPE="linux" ;;
    Darwin*)    OS_TYPE="macos" ;;
    CYGWIN*|MINGW*|MSYS*) OS_TYPE="windows" ;;
    *)          echo "未知操作系统: $OS"; exit 1 ;;
esac

echo "当前操作系统: $OS_TYPE"

# 创建输出目录
mkdir -p "$OUTPUT_DIR"

echo "开始编译..."

# 1. OpenWrt (x86_64) - Linux + AMD64
echo "  [-] OpenWrt (x86_64)..."
if [ "$OS_TYPE" = "macos" ]; then
    CC=x86_64-openwrt-linux-gcc GOOS=linux GOARCH=amd64 \
    go build -tags client -o "$OUTPUT_DIR/client_flow_collect_openwrt" -ldflags="-s -w" "$CLIENT_DIR"
else
    GOOS=linux GOARCH=amd64 \
    go build -tags client -o "$OUTPUT_DIR/client_flow_collect_openwrt" -ldflags="-s -w" "$CLIENT_DIR"
fi
echo "  ✓ OpenWrt 编译完成"

# 2. Windows (AMD64)
echo "  [-] Windows (AMD64)..."
GOOS=windows GOARCH=amd64 \
go build -tags client -o "$OUTPUT_DIR/client_flow_collect_windows.exe" -ldflags="-s -w" "$CLIENT_DIR"
echo "  ✓ Windows 编译完成"

# 3. Android (ARM64)
echo "  [-] Android (ARM64)..."
CGO_ENABLED=0 GOOS=android GOARCH=arm64 \
go build -tags client -o "$OUTPUT_DIR/client_flow_collect_android" -ldflags="-s -w" "$CLIENT_DIR"
echo "  ✓ Android 编译完成"

echo ""
echo "===== 编译全部完成 ====="
echo "输出目录: $OUTPUT_DIR"
ls -lh "$OUTPUT_DIR"
