#!/bin/bash
# client_build.sh - FlowCollect 客户端交互式纯静态编译脚本
# 所有目标均采用 CGO_ENABLED=0 产出纯静态可执行文件

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
CLIENT_DIR="$SCRIPT_DIR/client"
OUTPUT_DIR="$CLIENT_DIR/bin"
HOME_DIR="$(dirname "$SCRIPT_DIR")"

# ── 颜色定义 ──
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# ── 编译函数 ──

build_target() {
    local goos="$1"
    local goarch="$2"
    local output_name="$3"
    local label="$4"

    echo -e "${CYAN}  [-] ${label}...${NC}"
    CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" \
        go build -tags client -o "$OUTPUT_DIR/$output_name" -ldflags="-s -w" "$CLIENT_DIR"
    echo -e "${GREEN}  ✓ ${label} 编译完成${NC}"
}

# ── 智能路径探测 ──

print_post_build_hints() {
    local built_desktop="$1"
    local built_android="$2"

    echo ""
    echo -e "${YELLOW}===== 部署建议 =====${NC}"

    # 桌面端：检查 ~/clash-verge-rev
    if [ "$built_desktop" = "1" ] && [ -d "$HOME_DIR/clash-verge-rev" ]; then
        local sidecar_dir="$HOME_DIR/clash-verge-rev/src-tauri/sidecar"
        echo -e "${CYAN}[桌面端] 检测到宿主工程: ~/clash-verge-rev${NC}"
        echo -e "${CYAN}  推荐将产物移入 sidecar 目录以供 Tauri 打包:${NC}"
        echo ""
        echo "  # Windows"
        echo "  mv $OUTPUT_DIR/flow_collect_client_windows_amd64.exe $sidecar_dir/flow_collect_client-x86_64-pc-windows-msvc.exe"
        echo ""
        echo "  # macOS Intel"
        echo "  mv $OUTPUT_DIR/flow_collect_client_darwin_amd64 $sidecar_dir/flow_collect_client-x86_64-apple-darwin"
        echo ""
        echo "  # macOS Apple Silicon"
        echo "  mv $OUTPUT_DIR/flow_collect_client_darwin_arm64 $sidecar_dir/flow_collect_client-aarch64-apple-darwin"
        echo ""
    fi

    # Android 端：检查 ~/box_for_magisk
    if [ "$built_android" = "1" ] && [ -d "$HOME_DIR/box_for_magisk" ]; then
        local magisk_bin="$HOME_DIR/box_for_magisk/module/system/bin"
        echo -e "${CYAN}[Android] 检测到宿主工程: ~/box_for_magisk${NC}"
        echo -e "${CYAN}  推荐将产物移入模块 bin 目录并重命名:${NC}"
        echo ""
        echo "  # ARM64 (主力)"
        echo "  cp $OUTPUT_DIR/flow_collect_client_android_arm64 $magisk_bin/flow_collect_client"
        echo ""
        echo "  # AMD64 (模拟器/测试)"
        echo "  cp $OUTPUT_DIR/flow_collect_client_android_amd64 $magisk_bin/flow_collect_client"
        echo ""
    fi

    # 如果两个宿主都不存在
    if [ "$built_desktop" = "1" ] && [ "$built_android" = "1" ] && \
       [ ! -d "$HOME_DIR/clash-verge-rev" ] && [ ! -d "$HOME_DIR/box_for_magisk" ]; then
        echo -e "${YELLOW}  未检测到宿主工程目录 (clash-verge-rev / box_for_magisk)${NC}"
        echo -e "${YELLOW}  产物已保存在: $OUTPUT_DIR/${NC}"
    fi
}

# ── 编译指定目标 ──

do_build() {
    local choice="$1"
    local built_desktop=0
    local built_android=0

    mkdir -p "$OUTPUT_DIR"
    echo ""
    echo -e "${CYAN}开始编译 (纯静态 CGO_ENABLED=0)...${NC}"
    echo ""

    case "$choice" in
        1)  # 全部平台
            build_target "windows" "amd64"   "flow_collect_client_windows_amd64.exe" "Windows (AMD64)"
            build_target "darwin"  "amd64"   "flow_collect_client_darwin_amd64"      "macOS (Intel)"
            build_target "darwin"  "arm64"   "flow_collect_client_darwin_arm64"      "macOS (Apple Silicon)"
            build_target "android" "arm64"   "flow_collect_client_android_arm64"     "Android (ARM64)"
            build_target "android" "amd64"   "flow_collect_client_android_amd64"     "Android (AMD64)"
            built_desktop=1
            built_android=1
            ;;
        2)  # 仅 Windows
            build_target "windows" "amd64"   "flow_collect_client_windows_amd64.exe" "Windows (AMD64)"
            built_desktop=1
            ;;
        3)  # 仅 macOS Intel
            build_target "darwin"  "amd64"   "flow_collect_client_darwin_amd64"      "macOS (Intel)"
            built_desktop=1
            ;;
        4)  # 仅 macOS ARM
            build_target "darwin"  "arm64"   "flow_collect_client_darwin_arm64"      "macOS (Apple Silicon)"
            built_desktop=1
            ;;
        5)  # 仅 Android ARM64
            build_target "android" "arm64"   "flow_collect_client_android_arm64"     "Android (ARM64)"
            built_android=1
            ;;
        6)  # 仅 Android AMD64
            build_target "android" "amd64"   "flow_collect_client_android_amd64"     "Android (AMD64)"
            built_android=1
            ;;
    esac

    echo ""
    echo -e "${GREEN}===== 编译完成 =====${NC}"
    echo -e "输出目录: ${OUTPUT_DIR}"
    ls -lh "$OUTPUT_DIR"/flow_collect_client_*
    echo ""

    print_post_build_hints "$built_desktop" "$built_android"
}

# ── 交互菜单 ──

echo ""
echo -e "${CYAN}╔══════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║   FlowCollect 客户端编译工具 (纯静态)    ║${NC}"
echo -e "${CYAN}╠══════════════════════════════════════════╣${NC}"
echo -e "${CYAN}║  1) 编译全部平台                         ║${NC}"
echo -e "${CYAN}║  2) 仅编译 Windows (AMD64)               ║${NC}"
echo -e "${CYAN}║  3) 仅编译 macOS (Intel)                 ║${NC}"
echo -e "${CYAN}║  4) 仅编译 macOS (Apple Silicon)         ║${NC}"
echo -e "${CYAN}║  5) 仅编译 Android (ARM64)               ║${NC}"
echo -e "${CYAN}║  6) 仅编译 Android (AMD64)               ║${NC}"
echo -e "${CYAN}║  0) 退出                                 ║${NC}"
echo -e "${CYAN}╚══════════════════════════════════════════╝${NC}"
echo ""

read -rp "请选择编译目标 [0-6]: " choice

case "$choice" in
    0)
        echo "已退出。"
        exit 0
        ;;
    [1-6])
        do_build "$choice"
        ;;
    *)
        echo -e "${RED}无效选项: $choice${NC}"
        exit 1
        ;;
esac
