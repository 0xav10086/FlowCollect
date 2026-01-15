#!/bin/bash

# push_sync.sh - 自动化提交并同步至双仓库 (Public & Private)

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

echo -e "${CYAN}🚀 开始自动化提交与双向同步...${NC}"

# 1. 准备提交信息
timestamp=$(date "+%Y-%m-%d %H:%M:%S")
read -p "请输入提交信息 [默认: Auto-commit $timestamp]: " commit_msg
if [ -z "$commit_msg" ]; then
    commit_msg="Auto-commit $timestamp"
fi

# 2. 本地基础提交 (遵循 .gitignore，用于公开仓库)
echo -e "\n${YELLOW}[1/3] 正在执行本地基础提交...${NC}"
git add .
git commit -m "$commit_msg"

# --- 阶段 1: 推送到公开仓库 ---
echo -e "\n${YELLOW}[2/3] 正在推送到公开仓库 (Public)...${NC}"
git push public main

# --- 阶段 2: 全量备份到私密仓库 (包含 .ini 等敏感文件) ---
echo -e "\n${YELLOW}[3/3] 正在准备私密仓库全量备份...${NC}"

# A. 临时失效 .gitignore
if [ -f .gitignore ]; then
    mv .gitignore .gitignore.bak
fi

# 定义清理函数
cleanup() {
    if [ -f .gitignore.bak ]; then
        mv .gitignore.bak .gitignore
    fi
    git rm -r --cached . > /dev/null 2>&1
    git add .
    echo -e "${CYAN}🛡️ 环境已恢复，敏感文件重新进入忽略状态。${NC}"
}
trap cleanup EXIT

# B. 强制添加并创建临时备份提交
git add .
git commit -m "Private Backup: $commit_msg ($timestamp)"

# C. 强制推送到私密仓库
echo -e "正在全量推送到私密仓库..."
git push private main -f

# D. 撤销临时提交，回退到基础提交状态
git reset --soft HEAD~1
git restore --staged .

echo -e "\n${GREEN}✅ 所有操作已完成！代码已同步至双端。${NC}"