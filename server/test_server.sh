#!/usr/bin/env bash
# FlowCollect 服务端本地测试脚本
# 用法: 先启动服务端 (./flow_server_linux)，然后运行本脚本 (./test_server.sh)
# 前提: 需要 curl 和 jq (jq 可选，用于美化 JSON 输出)

set -euo pipefail

BASE_URL="${1:-http://localhost:8686}"
PASS=0
FAIL=0

check() {
    local desc="$1"
    local expected="$2"
    local actual="$3"

    if echo "$actual" | grep -q "$expected"; then
        echo "  ✅ PASS: $desc"
        ((PASS++))
    else
        echo "  ❌ FAIL: $desc"
        echo "    期望包含: $expected"
        echo "    实际响应: $(echo "$actual" | head -5)"
        ((FAIL++))
    fi
}

echo "=========================================="
echo " FlowCollect 服务端测试"
echo " 目标: $BASE_URL"
echo "=========================================="
echo ""

# ─────────────────────────────────────────
# Test 1: GET /sub — 动态订阅分发
# ─────────────────────────────────────────
echo "[Test 1] GET /sub — 动态订阅分发"
RESPONSE=$(curl -s -w "\n%{http_code}" "$BASE_URL/sub" 2>&1) || true
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')

check "HTTP 状态码为 200" "200" "$HTTP_CODE"
check "响应包含 Clash 基础配置 (port)" "port:" "$BODY"
check "响应包含 proxies 段" "proxies:" "$BODY"
check "响应包含 proxy-groups 段" "proxy-groups:" "$BODY"
check "响应包含 rules 段" "rules:" "$BODY"
check "响应包含 DNS 配置" "dns:" "$BODY"
check "响应包含 mixed-port" "mixed-port:" "$BODY"
echo ""

# ─────────────────────────────────────────
# Test 2: GET /sub?device=xxx — 带设备参数
# ─────────────────────────────────────────
echo "[Test 2] GET /sub?device=test-device — 带设备参数"
RESPONSE=$(curl -s -w "\n%{http_code}" "$BASE_URL/sub?device=test-device" 2>&1) || true
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')

check "HTTP 状态码为 200" "200" "$HTTP_CODE"
check "响应包含 FlowCollect 标识注释" "FlowCollect Dynamic Subscription" "$BODY"
echo ""

# ─────────────────────────────────────────
# Test 3: OPTIONS / — CORS 预检请求
# ─────────────────────────────────────────
echo "[Test 3] OPTIONS /api/stats — CORS 预检请求"
RESPONSE=$(curl -s -D - -o /dev/null -X OPTIONS \
    -H "Origin: https://dash.example.com" \
    -H "Access-Control-Request-Method: GET" \
    -H "Access-Control-Request-Headers: Authorization, Upgrade, Sec-WebSocket-Key" \
    "$BASE_URL/api/stats" 2>&1) || true

check "包含 Access-Control-Allow-Origin 头" "Access-Control-Allow-Origin:" "$RESPONSE"
check "包含 Upgrade 头放行" "Upgrade" "$RESPONSE"
check "包含 Sec-WebSocket-Key 头放行" "Sec-WebSocket-Key" "$RESPONSE"
check "包含 CF-Connecting-IP 头放行" "CF-Connecting-IP" "$RESPONSE"
check "HTTP 状态码为 204" "204" "$RESPONSE"
echo ""

# ─────────────────────────────────────────
# Test 4: GET /sub Content-Type 验证
# ─────────────────────────────────────────
echo "[Test 4] GET /sub Content-Type 验证"
CONTENT_TYPE=$(curl -s -o /dev/null -D - "$BASE_URL/sub" 2>&1 | grep -i "content-type" || true)

check "Content-Type 为 text/yaml" "text/yaml" "$CONTENT_TYPE"
echo ""

# ─────────────────────────────────────────
# 结果汇总
# ─────────────────────────────────────────
echo "=========================================="
echo " 测试结果: ✅ $PASS passed, ❌ $FAIL failed"
echo "=========================================="

if [ "$FAIL" -gt 0 ]; then
    exit 1
fi
