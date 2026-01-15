#!/bin/bash

# pull_sync.sh - 一键拉取并修复环境

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

echo -e "${CYAN}🔄 开始从私密仓库同步环境...${NC}"

# 1. 从私密仓库拉取最新代码 (包含 .ini 配置)
echo -e "\n${YELLOW}[1/2] 正在拉取全量数据...${NC}"
git pull private main

# 2. 修复环境索引
# 即使 .ini 文件已被拉取到本地，我们也需要让 Git 重新识别 .gitignore 规则
# 从而确保这些文件在本地开发时依然处于“忽略”状态，防止误提交到 public
echo -e "\n${YELLOW}[2/2] 正在修复本地 Git 索引...${NC}"
git rm -r --cached . > /dev/null 2>&1
git add .

echo -e "\n${GREEN}✅ 环境同步成功！${NC}"
echo -e "${CYAN}提示: 您现在的本地已包含最新的 .ini 配置文件，且 Git 已自动将其设为忽略状态。${NC}"