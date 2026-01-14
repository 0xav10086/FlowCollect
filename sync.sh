#!/bin/bash

# sync.sh - 跨平台双仓库同步脚本 (Windows/macOS)

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # 无颜色

echo -e "${CYAN}🚀 开始跨平台双仓库同步流程...${NC}"

# 1. 检查是否有未提交的代码
if [ -n "$(git status --porcelain)" ]; then
    echo -e "${RED}⚠️ 错误: 请先提交本地更改后再运行同步脚本！${NC}"
    exit 1
fi

# --- 阶段 1: 推送到公开仓库 ---
echo -e "\n${YELLOW}[1/2] 正在同步到公开仓库 (Public)...${NC}"
git push public main

# --- 阶段 2: 全量备份到私密仓库 ---
echo -e "\n${YELLOW}[2/2] 正在准备私密仓库全量备份...${NC}"

# A. 临时失效 .gitignore (兼容 Win/Mac 的 mv 命令)
if [ -f .gitignore ]; then
    mv .gitignore .gitignore.bak
fi

# 使用 try-finally 的逻辑（Shell 中使用 trap 捕获退出）
cleanup() {
    if [ -f .gitignore.bak ]; then
        mv .gitignore.bak .gitignore
    fi
    # 清除缓存并恢复环境
    git rm -r --cached . > /dev/null 2>&1
    git add .
    echo -e "${CYAN}🛡️ 环境已恢复，私密文件重新进入忽略状态。${NC}"
}
trap cleanup EXIT

# B. 强制添加所有文件
git add .

# C. 创建临时备份提交
timestamp=$(date "+%Y-%m-%d %H:%M:%S")
git commit -m "Private Backup: $timestamp"

# D. 强制推送到私密仓库
echo -e "正在强制推送到私密仓库..."
git push private main -f

# E. 撤销临时提交，回到干净状态
git reset --soft HEAD~1
git restore --staged .

echo -e "\n${GREEN}✅ 私密仓库全量备份完成！${NC}"