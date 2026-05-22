# FlowCollect 服务端子系统法典

> **Agent 读取指令**：接管服务端相关任务前，必须先阅读本文件。
> 本文档定义了 Go 服务端的架构目标、工作区目录规范、Gin 框架配置及安全策略。

## 1. 服务端架构总览

服务端以 **工作区目录** 形式部署，Go 后端承担两大核心职责：

1. **动态订阅分发枢纽**：读取同级目录下的 `*.yaml` 节点配置模板和 `RuleSet/` 规则集，通过 HTTP 路由（如 `/sub`）为客户端动态计算并下发 Clash 订阅配置。
2. **流量审计 API**：提供 REST API 和 WebSocket 端点，接收 Sidecar 上报的流量数据，存储至 SQLite，并向前端仪表盘实时推送。

后台脚本（如 `auto_update_node_and_rule.sh`）可独立维护 `RuleSet/` 和 `*.yaml` 配置，Go 后端实时读取最新状态，无需重启或重新编译。

### 1.1 架构目标

| 目标 | 说明 | 状态 |
|------|------|------|
| 动态订阅分发 | 读取 `*.yaml` + `RuleSet/` + `*.csv`，通过 `/sub` 路由生成个性化订阅链接 | `[x]` |
| 流量审计 API | REST + WebSocket 接收 Sidecar 上报数据 | `[x]` |
| SQLite 持久化 | 流量数据本地存储 | `[x]` |
| CORS + WebSocket 放行 | 允许跨域升级请求 | `[x]` |
| TrustedPlatform 真实 IP | 适配 Cloudflare 隧道部署 | `[x]` |

---

## 2. 工作区目录规范

服务端以目录形式提供，包含以下结构：

| 路径 | 职责 |
|------|------|
| `*.yaml` | Clash 核心节点配置模板（如 `bemly_node.yaml`、`cf_node.yaml`、`shanhuyun_node.yaml`） |
| `*.csv` | 规则集清单（`86_rule_set_collect.csv`） |
| `RuleSet/` | Clash 规则集文件目录 |
| `ServerSetting.ini` | 服务端运行时配置（Git 忽略） |
| `ServerSetting.ini.example` | 服务端配置模板 |
| `auto_update_node_and_rule.sh` | 自动化更新脚本（可选） |

Go 服务端只需读取目录内容运行，便于脚本后台动态更新配置而无需重启或重新编译。

---

## 3. 核心路由职责

### 3.1 订阅分发路由（`/sub`）

Go 后端读取同级目录下的 `*.yaml` 节点模板和 `RuleSet/` 规则集，根据客户端请求参数（如设备 ID、区域偏好）动态拼接完整的 Clash 配置文件，通过 HTTP 响应返回。

```
客户端 GET /sub?device=xxx
  → Go 后端读取 bemly_node.yaml + RuleSet/ + 86_rule_set_collect.csv
  → 动态计算并拼接完整 Clash 配置
  → 返回 text/yaml 响应
```

### 3.2 流量审计路由（`/api/*`）

REST API 接收 Sidecar 上报的流量数据，WebSocket 端点向前端仪表盘实时推送。

---

## 4. Gin 框架配置

### 4.1 CORS 放行 WebSocket 策略

服务端需配置 CORS 中间件，放行 WebSocket 升级请求。客户端通过 WebSocket 长连接上报流量数据，CORS 策略必须允许 `Upgrade: websocket` 头部通过。

### 4.2 TrustedPlatform 获取真实 IP

当服务端部署在 Cloudflare 隧道或反向代理之后时，需通过 Gin 的 `TrustedPlatform` 配置获取客户端真实 IP，避免所有请求被识别为代理服务器的 IP。
