#!/bin/bash
# 下载最新 GitHub Release 中的所有 .vsix 文件到当前目录
# 用法: ./scripts/download-latest-release.sh [目标目录]

set -e

# 配置
REPO="toheart/cocursor"
TARGET_DIR="${1:-.}"  # 默认下载到当前目录

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}=== 下载 CoCursor 最新 Release ===${NC}"
echo ""

# 检查 gh CLI 是否安装
if ! command -v gh &> /dev/null; then
    echo -e "${YELLOW}警告: GitHub CLI (gh) 未安装，将使用 curl 方式下载${NC}"
    USE_CURL=true
else
    USE_CURL=false
fi

# 创建目标目录
mkdir -p "$TARGET_DIR"
cd "$TARGET_DIR"

if [ "$USE_CURL" = true ]; then
    # 使用 curl + GitHub API 下载
    echo "正在获取最新 Release 信息..."
    
    LATEST_RELEASE=$(curl -s "https://api.github.com/repos/${REPO}/releases/latest")
    
    if [ -z "$LATEST_RELEASE" ] || echo "$LATEST_RELEASE" | grep -q "Not Found"; then
        echo -e "${RED}错误: 未找到任何 Release${NC}"
        echo "请确认仓库 ${REPO} 有可用的 Release"
        exit 1
    fi
    
    TAG_NAME=$(echo "$LATEST_RELEASE" | grep -o '"tag_name": "[^"]*"' | cut -d'"' -f4)
    echo -e "最新版本: ${GREEN}${TAG_NAME}${NC}"
    echo ""
    
    # 提取所有 .vsix 文件的下载链接
    DOWNLOAD_URLS=$(echo "$LATEST_RELEASE" | grep -o '"browser_download_url": "[^"]*\.vsix"' | cut -d'"' -f4)
    
    if [ -z "$DOWNLOAD_URLS" ]; then
        echo -e "${RED}错误: 最新 Release 中没有 .vsix 文件${NC}"
        exit 1
    fi
    
    echo "正在下载 .vsix 文件..."
    echo ""
    
    for url in $DOWNLOAD_URLS; do
        filename=$(basename "$url")
        echo -e "下载: ${YELLOW}${filename}${NC}"
        curl -L -o "$filename" "$url"
        echo -e "  ${GREEN}✓ 完成${NC}"
    done
else
    # 使用 gh CLI 下载
    echo "正在获取最新 Release..."
    
    LATEST_TAG=$(gh release list --repo "$REPO" --limit 1 --json tagName --jq '.[0].tagName')
    
    if [ -z "$LATEST_TAG" ]; then
        echo -e "${RED}错误: 未找到任何 Release${NC}"
        exit 1
    fi
    
    echo -e "最新版本: ${GREEN}${LATEST_TAG}${NC}"
    echo ""
    
    echo "正在下载所有 .vsix 文件..."
    gh release download "$LATEST_TAG" --repo "$REPO" --pattern "*.vsix"
fi

echo ""
echo -e "${GREEN}=== 下载完成 ===${NC}"
echo ""
echo "已下载的文件:"
ls -lh *.vsix 2>/dev/null || echo "  (无 .vsix 文件)"
echo ""
echo -e "${YELLOW}提示: 你可以在以下页面手动上传这些文件:${NC}"
echo "  https://marketplace.visualstudio.com/manage/publishers/tanglyan-cocursor"
