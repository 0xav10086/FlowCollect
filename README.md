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

## Architecture

### 数据流向

```
┌──────────────────┐      ┌──────────────┐      ┌──────────────────────────────┐
│  Client (Sidecar) │─────▶│  CF 隧道     │─────▶│  NAS / 内网 Server            │
│  ┌──────────────┐ │      │  (合规入口)  │      │  ┌────────────────────────┐  │
│  │ Clash Meta   │ │      └──────────────┘      │  │ Go + Gin               │  │
│  │ + FlowCollect│ │                            │  │ - REST API             │  │
│  │   Reporter   │ │                            │  │ - WebSocket 实时上报    │  │
│  └──────────────┘ │                            │  └────────────────────────┘  │
└──────────────────┘                             └──────────────┬───────────────┘
                                                               │
                                                               ▼
                                                 ┌──────────────────────────────┐
                                                 │  国内 VPS / CDN              │
                                                 │  ┌────────────────────────┐  │
                                                 │  │ Vue 3 SPA (smart_spend)│  │
                                                 │  │ + metacubexd 面板       │  │
                                                 │  └────────────────────────┘  │
                                                 └──────────────────────────────┘
```

### 设计理念

| 层级 | 策略 | 原因 |
|------|------|------|
| **后端** | 部署在 NAS / 内网，通过 Cloudflare 隧道暴露 | 避免直接暴露真实 IP，规避 VPS 封禁风险 |
| **前端** | 托管在国内 VPS / CDN | 合规备案，访问速度快 |
| **通信** | 全链路 HTTPS / WSS | 数据加密传输，CF 隧道自动 TLS |

---

## Monorepo Structure

```
FlowCollect/
├── client/                  # Sidecar 客户端 (Go)
│   ├── main.go              # 入口：流量采集 + WebSocket 上报
│   ├── config.yaml.example  # 配置模板（x-flow-collect 扩展字段）
│   └── agent.md             # 客户端子系统法典（AI Agent 必读）
│
├── server/                  # 服务端 (Go + Gin)
│   ├── main.go              # 入口：REST API + 静态资源托管
│   ├── web/                 # 前端构建产物（运行时目录）
│   └── agent.md             # 服务端子系统法典（AI Agent 必读）
│
├── smart_spend/             # 前端 (Vue 3 + Vite + Element Plus)
│   ├── src/
│   │   ├── views/           # 页面组件
│   │   └── utils/           # HTTP/WS 工具模块
│   ├── .env.development     # 开发环境变量
│   ├── .env.production      # 生产环境变量
│   └── agent.md             # 前端子系统法典（AI Agent 必读）
│
├── AGENTS.md                # 全局架构宪法（AI Agent 核心知识库）
├── ROADMAP.md               # 项目进度与 Release 流水线
└── .github/workflows/
    └── release.yml          # 全自动 Release CI
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
curl -sL https://github.com/your-org/FlowCollect/releases/latest/download/flow_collect.tgz -o flow_collect.tgz

# 解压并启动
mkdir -p /opt/flow_collect && tar xzf flow_collect.tgz -C /opt/flow_collect/
cd /opt/flow_collect && ./flow_server_linux
```

### 2. 客户端配置

客户端通过 Clash Meta 配置文件的 `x-flow-collect` 扩展字段读取连接信息：

```yaml
# config.yaml (Mihomo / Clash Meta)
x-flow-collect:
  remote-server: "wss://your-domain.com/ws/traffic"
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
VITE_API_BASE_URL=https://your-domain.com     # 生产
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
