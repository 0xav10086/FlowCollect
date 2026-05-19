# FlowCollect 全局架构宪法 (Agent 核心知识库)

> **Agent 读取指令**：
> 本文档定义了 FlowCollect 的宏观演进目标、架构规范和核心禁忌。
> **接管具体子系统任务前，必须先读取对应的子系统法典（见 §4 路由表）。**
> **严格禁止**试图一次性完成所有任务。必须按照人类分配的单一子任务进行局部迭代。

---

## 1. 终极愿景与宏观拓扑

FlowCollect 正在从一个被动的分布式流量审计系统升级为深度集成的代理与审计一体化平台。

**核心理念**：客户端采用 **旁路注入（Sidecar）** 上报模式，服务端采用 **工作区目录** 打包部署。

### 数据流向

```
Client (Sidecar) ──▶ CF 隧道 ──▶ Server (Go + Gin) ──▶ Frontend (Vue 3 SPA)
```

### 工作区目录职责边界

| 目录 | 职责（一句话） |
|------|---------------|
| `client/` | 唯一合法的客户端 Sidecar（Go 语言）源码及编译输出目录。 |
| `server/` | 唯一合法的服务端（Go 语言）源码及编译输出目录。 |
| `smart_spend/` | 纯粹的 Vue 3 前端工程。`dist/` 只允许存放前端静态资源，绝不允许污染跨平台二进制。 |

---

## 2. 绝对禁忌 (Red Lines)

以下规则在任何情况下不得违反：

1. **禁止硬编码内网 IP**：所有网络地址通过 `.yaml` 配置文件读取，通过结构体抽象传递。
2. **禁止明文代理节点信息**：`secret`、`token` 等敏感信息不得出现在源码、日志或前端持久化存储中。
3. **禁止跨目录污染**：`client/` 的编译产物不得放入 `smart_spend/dist/`，`server/` 的二进制不得混入前端目录。
4. **禁止配置硬编码**：所有配置通过 `.yaml` 文件或环境变量注入，代码中不得出现魔法值。
5. **模块化边界**：严格维护 `client`、`server`、`smart_spend` 三个目录的隔离，各自拥有独立的 `agent.md` 法典。

---

## 3. 状态确认机制

Agent 在修改任何关键文件后，必须输出测试方案，等待人类反馈"测试通过"后，才能推进到下一步。

---

## 4. 子系统法典索引 (路由表)

> **重要**：接管具体任务前，必须先读取对应目录下的 `agent.md`。

| 子系统 | 法典路径 | 涵盖内容 |
|--------|----------|----------|
| **客户端**（Android / 桌面） | `client/agent.md` | box_for_magisk 集成、clash-verge-rev 集成、交叉编译、Sidecar 进程共管、端到端测试 |
| **服务端**（Go + Gin） | `server/agent.md` | 工作区目录部署、Gin 配置、CORS WebSocket 放行、TrustedPlatform 真实 IP |
| **前端**（Vue 3 SPA） | `smart_spend/agent.md` | metacubexd iframe 嵌入、环境变量规范、安全约束 |

---

## 5. 三大开源项目 Fork 集成概览

| 项目 | 角色 | 状态 | 详细方案 |
|------|------|------|----------|
| MetaCubeX/metacubexd | 前端面板嵌入 | `[x] 已完成` | → `smart_spend/agent.md` |
| taamarin/box_for_magisk | Android 客户端宿主 | `[x] Step 1-5 完成` | → `client/agent.md` |
| clash-verge-rev/clash-verge-rev | 桌面客户端宿主 | `[ ] 待执行` | → `client/agent.md` |

---

## 6. 全局发布（Release）流水线

### 架构总览

```
上游源头                    核心中枢                     数据平面
───────────                ──────────                  ──────────
metacubexd ──┐
             ├──▶ FlowCollect (Tag Release) ──▶ clash-verge-rev (桌面端)
             │         │                         box_for_magisk (移动端)
             │         ▼
             │    flow_collect.tgz (VPS 部署包)
             │         │
             └─────────┘
```

### 6.1 FlowCollect 仓库 — Release CI `[ ] 待执行`

**触发条件**：推送 `v*` Tag（如 `v1.0.0`）或手动 `workflow_dispatch`。

**工作流文件**：`.github/workflows/release.yml`

**执行步骤**：

1. **编译所有 Sidecar 客户端**（并行矩阵）：
   | 目标 | 环境变量 | 产出文件名 |
   |------|---------|-----------|
   | Windows AMD64 | `GOOS=windows GOARCH=amd64` | `flow_collect_client_windows_amd64.exe` |
   | macOS Intel | `GOOS=darwin GOARCH=amd64` | `flow_collect_client_darwin_amd64` |
   | macOS ARM | `GOOS=darwin GOARCH=arm64` | `flow_collect_client_darwin_arm64` |
   | Linux AMD64 | `GOOS=linux GOARCH=amd64` | `flow_collect_client_linux_amd64` |
   | Android ARM64 | `GOOS=android GOARCH=arm64` | `flow_collect_client_android_arm64` |
   | Android AMD64 | `GOOS=android GOARCH=amd64` | `flow_collect_client_android_amd64` |

   全部采用 `CGO_ENABLED=0` 纯静态编译。

2. **构建 VPS 服务端资源包** (`flow_collect.tgz`)：
   - 编译 `server/` → `flow_server_linux`（`CGO_ENABLED=0 GOOS=linux GOARCH=amd64`）
   - 构建 `smart_spend/` → `dist/`（`npm ci && npm run build`）
   - 下载 metacubexd 最新 Release 的 `compressed-dist.tgz` 并解压
   - 目录组装：
     ```
     flow_collect_staging/
     ├── flow_server_linux        # 服务端可执行文件
     └── web/                     # 所有前端静态资源
         ├── index.html           # smart_spend 构建产物
         ├── js/
         ├── css/
         └── node-panel/          # metacubexd 面板产物
     ```
   - 打包：`tar czf flow_collect.tgz -C flow_collect_staging .`

3. **发布到 GitHub Releases**：
   - 上传所有客户端二进制 + `flow_collect.tgz`
   - 生成 Release Notes（含平台对照表和部署说明）

**发布命令**：
```bash
cd ~/FlowCollect
git tag v1.0.0
git push origin v1.0.0
```

### 6.2 clash-verge-rev 仓库 — 消费 FlowCollect Release `[ ] 待执行`

**改造目标**：修改 `scripts/prebuild.mjs`，在 Tauri build 之前从 FlowCollect Release 下载对应架构的客户端二进制。

**逻辑**：
1. 获取 FlowCollect 最新 Release tag
2. 下载当前平台对应的 `flow_collect_client_<os>_<arch>[.exe]`
3. 放置到 `src-tauri/sidecar/` 目录，命名为 `flow_collect_client-<target_triple>[.exe]`
4. Tauri build 自动将其打包进安装包

**触发方式**：clash-verge-rev 自身的 CI 或手动触发。

### 6.3 box_for_magisk 仓库 — 消费 FlowCollect Release `[ ] 待执行`

**改造目标**：在 Magisk 模块打包 CI 中，自动从 FlowCollect Release 下载 Android ARM64 二进制。

**逻辑**：
1. 获取 FlowCollect 最新 Release tag
2. 下载 `flow_collect_client_android_arm64`
3. 放入模块 `bin/` 目录，重命名为 `flow_collect_client`
4. 正常执行模块 zip 打包

### 6.4 metacubexd 仓库 — 上游依赖 `[x] 已有`

**已有机制**：metacubexd 自身的 GitHub Actions 在发布 Release 时自动编译并提供 `compressed-dist.tgz` 静态资源包。

**FlowCollect 消费方式**：在 Release CI 中通过 GitHub API 获取最新 Release URL 并下载。
