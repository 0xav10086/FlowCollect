# FlowCollect 项目进度与发布流水线

> 本文档记录三大开源项目 Fork 集成状态与全局 Release 流水线。

---

## 1. 三大开源项目 Fork 集成概览

| 项目 | 角色 | 状态 | 详细方案 |
|------|------|------|----------|
| MetaCubeX/metacubexd | 前端面板嵌入 | `[x] 已完成` | → `smart_spend/agent.md` |
| taamarin/box_for_magisk | Android 客户端宿主 | `[x] Step 1-5 完成` | → `client/agent.md` |
| clash-verge-rev/clash-verge-rev | 桌面客户端宿主 | `[ ] 待执行` | → `client/agent.md` |

---

## 2. 全局发布（Release）流水线

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

### 2.1 FlowCollect 仓库 — Release CI `[ ] 待执行`

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

### 2.2 clash-verge-rev 仓库 — 消费 FlowCollect Release `[ ] 待执行`

**改造目标**：修改 `scripts/prebuild.mjs`，在 Tauri build 之前从 FlowCollect Release 下载对应架构的客户端二进制。

**逻辑**：
1. 获取 FlowCollect 最新 Release tag
2. 下载当前平台对应的 `flow_collect_client_<os>_<arch>[.exe]`
3. 放置到 `src-tauri/sidecar/` 目录，命名为 `flow_collect_client-<target_triple>[.exe]`
4. Tauri build 自动将其打包进安装包

**触发方式**：clash-verge-rev 自身的 CI 或手动触发。

### 2.3 box_for_magisk 仓库 — 消费 FlowCollect Release `[ ] 待执行`

**改造目标**：在 Magisk 模块打包 CI 中，自动从 FlowCollect Release 下载 Android ARM64 二进制。

**逻辑**：
1. 获取 FlowCollect 最新 Release tag
2. 下载 `flow_collect_client_android_arm64`
3. 放入模块 `bin/` 目录，重命名为 `flow_collect_client`
4. 正常执行模块 zip 打包

### 2.4 metacubexd 仓库 — 上游依赖 `[x] 已有`

**已有机制**：metacubexd 自身的 GitHub Actions 在发布 Release 时自动编译并提供 `compressed-dist.tgz` 静态资源包。

**FlowCollect 消费方式**：在 Release CI 中通过 GitHub API 获取最新 Release URL 并下载。
