# FlowCollect 服务端子系统法典

> **Agent 读取指令**：接管服务端相关任务前，必须先阅读本文件。
> 本文档定义了 Go 服务端的部署架构、工作区目录规范、Gin 框架配置及安全策略。

## 1. 服务端架构总览

服务端以 **工作区目录** 形式部署，Go 后端负责提供 API 并映射 `web/` 目录下的静态资源。
后台脚本（如 `auto_update_node_and_rule.sh`）可以独立维护 `RuleSet/` 和配置，Go 后端实时读取最新状态。

### 1.1 架构目标 `[x] 已完成`

通过分离 Go 服务与静态资源目录，实现高度灵活的热更新。Go 后端负责提供 API 并映射 `web/` 目录下的静态资源。

---

## 2. 工作区目录规范

服务端以目录形式提供，包含以下结构：

| 路径 | 职责 |
|------|------|
| `web/` | 存放所有的前端静态资源，包括主框架 `smart_spend` 的构建产物，以及按需加载的定制版 `metacubexd` 面板 |
| `*.yaml` | Clash 核心配置文件 |
| `RuleSet/` | Clash 规则集目录 |
| `86_rule_set_collect.csv` | 规则集清单 |
| `auto_update_node_and_rule.sh` | 自动化更新脚本 |

Go 服务端只需读取目录内容运行，便于脚本后台动态更新配置而无需重启或重新编译。

---

## 3. Gin 框架配置

### 3.1 静态资源托管

Go 后端通过 Gin 的 `StaticFS` 映射 `web/` 目录，为前端 SPA 提供静态资源服务。

### 3.2 CORS 放行 WebSocket 策略

服务端需配置 CORS 中间件，放行 WebSocket 升级请求。客户端通过 WebSocket 长连接上报流量数据，CORS 策略必须允许 `Upgrade: websocket` 头部通过。

### 3.3 TrustedPlatform 获取真实 IP

当服务端部署在 Cloudflare 隧道或反向代理之后时，需通过 Gin 的 `TrustedPlatform` 配置获取客户端真实 IP，避免所有请求被识别为代理服务器的 IP。

---

## 4. 前端资源整合（web/ 目录）

`web/` 目录的组装由全局 Release 流水线（见 `AGENTS.md` §6）完成：

```
web/
├── index.html           # smart_spend 构建产物
├── js/
├── css/
└── node-panel/          # metacubexd 面板产物
```

服务端无需关心 `web/` 的构建过程，只需确保 Gin StaticFS 正确指向该目录。
