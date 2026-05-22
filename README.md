<p align="center">
  <h1 align="center">FlowCollect</h1>
  <p align="center">
    <b>代理与审计一体化平台 | Sidecar 旁路注入 | 混合云架构</b>
  </p>
  <p align="center">
    <img src="https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go&logoColor=white" alt="Go">
    <img src="https://img.shields.io/badge/Vue-3-4FC08D?logo=vue.js&logoColor=white" alt="Vue 3">
    <img src="https://img.shields.io/badge/Vite-5-646CFF?logo=vite&logoColor=white" alt="Vite">
    <img src="https://img.shields.io/badge/GitHub%20Actions-CI/CD-2088FF?logo=githubactions&logoColor=white" alt="CI/CD">
    <img src="https://img.shields.io/badge/License-MIT-blue" alt="License">
  </p>
</p>

---

**FlowCollect** 正在从一个被动的分布式流量审计系统，升级为深度集成的**代理与审计一体化平台**。

核心理念：客户端以 **Sidecar（旁路注入）** 模式无感上报流量，服务端以**工作区目录**打包部署，前端通过 Cloudflare 隧道合规托管 —— 实现「前端合规托管 + 后端穿透避险」的极客架构。

---

## Prerequisites

在部署 FlowCollect 之前，请确保以下基础设施已就绪：

| 组件 | 说明 | 备注 |
|------|------|------|
| **内网 / NAS 主机** | 运行 Go 服务端（`flow_server_linux`）的 Linux 设备 | 需要长期在线，建议使用 NAS 或小型服务器 |
| **Cloudflare Zero Trust** | 账号 + Tunnel 配置 | 用于将内网 NAS 的 API/WS 端口安全暴露到公网，**不暴露真实 IP** |
| **国内 VPS / CDN** | 用于托管 Vue 3 前端静态资源 | 需完成 ICP 备案的域名，例如 `dash.your-domain.com` |
| **域名** | 两个子域名（可同域） | 一个指向 VPS（前端），一个指向 CF Tunnel（后端 API） |
| **Clash Meta / Mihomo** | 客户端代理内核 | 用于运行 Sidecar 注入的宿主环境 |

---

## Architecture

### 数据流向 — "Y 型"拓扑

用户浏览器是整个数据流的分叉点：静态资源走 VPS，动态数据走 CF 隧道直抵 NAS。**NAS 与 VPS 之间没有任何直接通信。**

```
                        ┌─────────────────────────────────┐
                        │         用户浏览器               │
                        │  (dash.0xav10086.space)          │
                        └────────┬──────────────┬─────────┘
                                 │              │
                    静态资源请求  │              │  API / WebSocket 请求
                 (HTML/JS/CSS)   │              │  (wss://api.0xav10086.space)
                                 │              │
                                 ▼              ▼
                  ┌──────────────────┐    ┌──────────────────────┐
                  │  国内 VPS        │    │  Cloudflare Tunnel   │
                  │                  │    │  (Zero Trust)        │
                  │  Nginx / CDN     │    │                      │
                  │  ┌────────────┐  │    └──────────┬───────────┘
                  │  │ Vue 3 SPA  │  │               │
                  │  │ (dist/)    │  │               │
                  │  └────────────┘  │               │
                  └──────────────────┘               │
                                                     ▼
                                          ┌──────────────────────┐
                                          │  NAS / 内网 Server    │
                                          │                      │
                                          │  Go + Gin            │
                                          │  ├── REST API        │
                                          │  ├── WebSocket 实时流 │
                                          │  └── SQLite 存储      │
                                          └──────────┬───────────┘
                                                     │
                                           Sidecar 上报 (WSS)
                                                     │
                                          ┌──────────▼───────────┐
                                          │  客户端 (Sidecar)     │
                                          │                      │
                                          │  Clash Meta / Mihomo │
                                          │  + FlowCollect       │
                                          │    Reporter          │
                                          └──────────────────────┘
```

**拆解说明**：

1. **静态资源路径**：用户浏览器访问 `dash.0xav10086.space` → 国内 VPS（Nginx）返回 Vue 3 SPA 的 `index.html`、JS、CSS 等静态文件。此路径不经过 CF 隧道。
2. **动态数据路径**：浏览器中运行的 Vue 3 SPA 通过 `wss://api.0xav10086.space`（或 `https://`）发起 API/WebSocket 请求 → 经 Cloudflare 隧道穿透至 NAS 内网的 Go 服务端。
3. **Sidecar 上报路径**：客户端代理内核（Clash Meta）旁路注入的 FlowCollect Reporter 通过 WSS 将流量数据上报至同一 CF 隧道端点。
4. **物理隔离**：NAS 与 VPS 之间**零通信**。浏览器是唯一的汇聚点，两条路径在用户侧合流，在服务侧完全隔离。

| 层级 | 策略 | 原因 |
|------|------|------|
| **后端** | 部署在 NAS / 内网，通过 Cloudflare 隧道暴露 | 避免直接暴露真实 IP，规避 VPS 封禁风险 |
| **前端** | 托管在国内 VPS / CDN | 合规备案，访问速度快 |
| **通信** | 全链路 HTTPS / WSS | 数据加密传输，CF 隧道自动 TLS |

---

## Monorepo Structure

```
FlowCollect/
├── client/                      # Sidecar 客户端 (Go)
│   ├── client.go                # 入口：流量采集 + WebSocket 上报
│   ├── go.mod / go.sum          # Go 依赖管理
│   ├── agent.md                 # 客户端子系统法典（AI Agent 必读）
│   └── bin/                     # 编译产物输出目录
│
├── server/                      # 服务端 (Go + Gin)
│   ├── main.go                  # 入口：HTTP Server 启动
│   ├── config.go                # 配置加载
│   ├── handlers.go              # REST API 路由处理
│   ├── websocket.go             # WebSocket 实时数据流
│   ├── service.go               # 业务逻辑层
│   ├── db.go                    # SQLite 数据库操作
│   ├── fake_api.go              # 仿真数据 API（开发/演示用）
│   ├── yaml_config.go           # YAML 配置解析
│   ├── utils.go                 # 工具函数
│   ├── email_test.go            # 邮件告警测试
│   ├── ServerSetting.ini        # 服务端运行时配置（Git 忽略）
│   ├── ServerSetting.ini.example# 服务端配置模板
│   ├── go.mod / go.sum          # Go 依赖管理
│   └── agent.md                 # 服务端子系统法典（AI Agent 必读）
│
├── smart_spend/                 # 前端 (Vue 3 + Vite + Element Plus)
│   ├── src/
│   │   ├── views/               # 页面组件
│   │   ├── router/              # 路由配置
│   │   ├── utils/               # HTTP/WS 工具模块
│   │   ├── assets/              # 静态资源
│   │   ├── App.vue              # 根组件
│   │   └── main.ts              # 入口文件
│   ├── metacubexd/              # metacubexd 面板集成
│   ├── dist/                    # 构建产物（部署到 VPS）
│   ├── .env.development         # 开发环境变量
│   ├── .env.production          # 生产环境变量
│   ├── vite.config.ts           # Vite 配置
│   ├── tailwind.config.js       # Tailwind CSS 配置
│   ├── package.json             # 前端依赖
│   └── agent.md                 # 前端子系统法典（AI Agent 必读）
│
├── client_build.sh              # 客户端跨平台编译脚本
├── server_build.sh              # 服务端 + 前端一键编译脚本
├── setting.ini.example          # 全局配置模板
├── go.work / go.work.sum        # Go Workspace 工作区
├── AGENTS.md                    # 全局架构宪法（AI Agent 核心知识库）
├── ROADMAP.md                   # 项目进度与 Release 流水线
└── .github/workflows/
    ├── release.yml              # 全自动 Release CI
    └── cl.yml.disabled          # 已禁用的旧 CI 配置
```

---

## Features

### 跨平台 Sidecar 编译

所有客户端二进制通过 `CGO_ENABLED=0` 纯静态编译，零依赖：

| 平台 | 架构 | 产出文件 |
|------|------|----------|
| Windows | AMD64 | `flow_collect_client_windows_amd64.exe` |
| macOS | Intel / Apple Silicon | `flow_collect_client_darwin_{amd64,arm64}` |
| Linux | AMD64 | `flow_collect_client_linux_amd64` |
| Android | ARM64 | `flow_collect_client_android_arm64` |

### GitHub Actions 全自动 Release

推一个 Tag，一切自动完成：

```bash
git tag v1.0.0
git push origin v1.0.0
```

CI 自动执行：
1. 并行编译所有平台 Sidecar 客户端
2. 构建 VPS 服务端资源包 (`flow_collect.tgz`)
3. 拉取上游 [metacubexd](https://github.com/MetaCubeX/metacubexd) 面板产物
4. 打包发布到 GitHub Releases

### Release 级联更新流水线

以下三个项目均为 [0xav10086](https://github.com/0xav10086) 的魔改 Fork，经过定制后实现了与 FlowCollect 的深度联动。

```
上游 (0xav10086/metacubexd)       核心中枢                     下游 (0xav10086/*)
───────────────────────          ──────────                  ──────────
0xav10086/metacubexd ──┐
                       ├──▶ FlowCollect (Tag Release) ──▶ 0xav10086/clash-verge-rev (桌面端)
                       │         │                         0xav10086/box_for_magisk (移动端)
                       │         ▼
                       │    flow_collect.tgz (VPS 部署包)
                       │         │
                       └─────────┘
```

**级联更新逻辑**：
- **上游更新**：`0xav10086/metacubexd` 发布新版本 -> FlowCollect Release CI 自动拉取其 `compressed-dist.tgz` 并集成到 `web/node-panel/`
- **核心发布**：FlowCollect 推送 Tag -> CI 编译所有平台 Sidecar + 打包 `flow_collect.tgz` -> 发布到 GitHub Releases
- **下游消费**：`0xav10086/clash-verge-rev` 和 `0xav10086/box_for_magisk` 的 CI 自动从 FlowCollect Release 拉取对应架构的客户端二进制并打包

| 项目 | 层级 | 角色 | 状态 |
|------|------|------|------|
| [0xav10086/metacubexd](https://github.com/0xav10086/metacubexd) | **上游** | 魔改版节点面板，FlowCollect 自动拉取其构建产物 | Done |
| [0xav10086/clash-verge-rev](https://github.com/0xav10086/clash-verge-rev) | **下游** | 魔改版桌面客户端，集成 FlowCollect Sidecar | WIP |
| [0xav10086/box_for_magisk](https://github.com/0xav10086/box_for_magisk) | **下游** | 魔改版 Android Magisk 模块，集成 FlowCollect Sidecar | Done |

---

## Quick Start

### 1. 服务端部署

```bash
# 下载最新 Release
curl -sL https://github.com/0xav10086/FlowCollect/releases/latest/download/flow_collect.tgz -o flow_collect.tgz

# 解压并启动
mkdir -p /opt/flow_collect && tar xzf flow_collect.tgz -C /opt/flow_collect/
cd /opt/flow_collect && ./flow_server_linux
```

### 2. 客户端配置

客户端通过 Clash Meta 配置文件的 `x-flow-collect` 扩展字段读取连接信息：

```yaml
# config.yaml (Mihomo / Clash Meta)
x-flow-collect:
  remote-server: "wss://api.your-domain.com/ws/traffic"
  remote-token: "your-secret-token"
  device-id: "my-device-01"
```

启动 Sidecar：

```bash
./flow_collect_client_linux_amd64 -c /path/to/config.yaml
```

### 3. 前端开发

```bash
cd smart_spend
npm install
npm run dev    # 开发服务器 -> http://localhost:8687
```

环境变量通过 `.env.development` / `.env.production` 注入：

```bash
VITE_API_BASE_URL=http://localhost:8080       # 开发
VITE_API_BASE_URL=https://api.your-domain.com # 生产
```

---

## Contributing / AI Agent Workflow

本项目采用**渐进式披露（Progressive Disclosure）**的 AI 协作模式。

### 对于人类贡献者

- Commit Message 严格遵循 [Angular 规范](https://www.conventionalcommits.org/en/v1.0.0/)：`feat(client): add xxx`、`fix(server): resolve xxx`
- 每个子系统（`client/`、`server/`、`smart_spend/`）独立迭代，禁止跨模块大杂烩提交

### 对于 AI Agent

> **强制规则**：接管任何子系统任务前，必须先读取对应目录下的 `agent.md`。

| 任务范围 | 必读文档 |
|----------|----------|
| Android / 桌面客户端 | `client/agent.md` |
| 服务端 API / 部署 | `server/agent.md` |
| 前端 UI / 构建 | `smart_spend/agent.md` |
| 全局架构 / 禁忌 | `AGENTS.md` |
| 项目进度 / Release | `ROADMAP.md` |

AI Agent 执行 `git commit` 时必须携带身份标识：

```bash
git commit --author="Claude AI <claude@anthropic.com>" -m "feat(scope): description"
```

---

## License

[MIT](LICENSE)
