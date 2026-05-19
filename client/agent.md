# FlowCollect 客户端子系统法典

> **Agent 读取指令**：接管客户端相关任务前，必须先阅读本文件。
> 本文档定义了 Android 端（box_for_magisk）和桌面端（clash-verge-rev）的集成架构与实施细节。

## 1. 客户端架构总览

客户端以 **Sidecar（旁路守护进程）** 模式存在，不是独立给用户双击运行的带界面软件。
它是被集成到魔改版的宿主客户端（桌面的 clash-verge-rev、Android 的 box_for_magisk）中。
宿主程序在拉起核心（如 Mihomo）时，会在后台静默拉起这个 Sidecar 进程，共同完成代理与数据上报。

### 1.1 Android 客户端架构：taamarin/box_for_magisk `[x] Step 1-5 完成`

- **目标**：实现底层的透明代理与无感上报。
- **集成思路**：在 Magisk/KernelSU 模块的启动脚本中，拉起 Clash Meta 代理内核的同时，拉起 FlowCollect 的审计上报进程（Go 编译的二进制文件）。两者需实现进程级共存。

### 1.2 桌面客户端架构：clash-verge-rev/clash-verge-rev `[ ] 待执行`

- **目标**：为现有 GUI 增加无感上报功能。
- **集成思路**：在 `clash-verge-rev` 原有的架构基础上，集成 FlowCollect 的上报模块，使其能在代理流量的同时将数据上报至服务端。

---

## 2. Android 客户端架构细纲

### 前置准备

工作区：`~/box_for_magisk`（独立 clone 的 taamarin/box_for_magisk 仓库）。

集成目标：在 Magisk/KernelSU 模块开机启动时，实现 Clash Meta 内核 + FlowCollect 审计上报进程的 Sidecar 共存。

### Step 1: Fork 分析 — 理解 box_for_magisk 模块结构

- **Agent 任务**：clone 或进入 `taamarin/box_for_magisk` 仓库，阅读其目录结构。
- **关键文件**：
  - `module.prop` — Magisk 模块元信息
  - `service.sh` — 模块启动脚本（通常是 boot_completed 后执行）
  - `post-fs-data.sh` — 更早期的初始化脚本
  - `clash.service` 或类似 — Clash 核心的进程管理脚本
- **产出物**：向人类汇报模块的启动流程、Clash 核心的启动参数（config.yaml 路径、端口等），以及放置附加二进制文件的目录规范。

### Step 2: 植入 FlowCollect 审计二进制 `[x] 已完成`

- **Agent 任务**：将 FlowCollect 的 Go 客户端编译为 Android ARM64 二进制，放入模块目录。
- **具体动作**：
  1. 在 FlowCollect 仓库中，确认 `client/` 目录的代码支持 Android 交叉编译（`GOOS=android GOARCH=arm64`）。
  2. 执行交叉编译，产出 `flow_collect_client_android`。
  3. 复制到 `box_for_magisk` 的合适目录（通常为 `<module>/bin/` 或 `<module>/data/`）。
- **验收标准**：二进制文件存在于模块目录，且 file 命令确认架构为 `ARM aarch64`。
- **已完成**：
  - 废弃 INI 配置，改用 YAML（`gopkg.in/yaml.v3`）
  - 支持 `-c` 命令行参数指定 `config.yaml` 路径
  - 支持 `FLOW_COLLECT_CONFIG` 环境变量
  - 智能路径检测：当前目录 → 可执行文件目录 → Android 默认路径
  - 解析 `x-flow-collect` 扩展字段（Mihomo 忽略未知顶层字段）
  - 编译指令：`GOOS=android GOARCH=arm64 go build -tags client -o flow_collect_client_android -ldflags="-s -w"`

### Step 3: 改写启动脚本 — Sidecar 进程共管 `[x] 已完成`

- **Agent 任务**：修改 `service.sh`（或模块的进程管理器脚本），在拉起 Clash Meta 核心的同时拉起 FlowCollect 客户端。
- **具体动作**：
  1. 在启动 Clash 核心的命令之后（或之前），增加启动 FlowCollect 客户端的逻辑。
  2. 确保 `flow_collect_client` 的配置参数（服务端地址、Token、设备 ID）通过环境变量或配置文件传递。
  3. 增加进程保活逻辑：Clash 核心退出时是否同时退出 collector？collector 崩溃时是否自动重启？
- **关键设计决策**：
  - 两个进程的命令空间（共享 netns？Clash 需要透明代理权限）
  - 日志输出策略（各自写独立文件，还是统一由 logd 管理）
- **验收标准**：模块安装后重启设备，`ps | grep flow_collect` 和 `ps | grep clash` 均能看到进程在运行。
- **已完成**：
  - 修改 `customize.sh`：在刷入时将 `flow_collect_client_android` 复制到 `/data/adb/box/bin/flow_collect_client`，设置 `root:net_admin` (0:3005) 权限
  - 修改 `box.service`：
    - `start_box()`：在 `box_bin_status` 之后启动 FlowCollect Sidecar，使用 `nohup busybox setuidgid root:net_admin` 运行，PID 写入 `${box_run}/flow_collect.pid`，日志输出到 `${box_run}/flow_collect.log`
    - `stop_box()`：在 `stop_cron` 之后、主进程 kill 之前，优雅停止 FlowCollect 客户端（SIGTERM），清理 PID 文件
  - 配置获取：通过 `-c /data/adb/box/clash/config.yaml` 显式指定（无环境变量）
  - 防劫持策略：使用 `root:net_admin` 运行，复用现有 iptables UID/GID 豁免规则

### Step 4: 流量上报验证 — 端到端测试

- **Agent 任务**：在真实的 Android 设备或模拟器上安装模块，验证数据链路。
- **具体动作**：
  1. 将模块打包为 `.zip`，通过 Magisk/KernelSU Manager 刷入。
  2. 确认 Clash 代理正常（能科学上网）。
  3. 确认 FlowCollect 服务端收到设备上报的流量数据（访问服务端 API `/api/stats` 能看到该设备）。
- **验收标准**：服务端设备列表中出现新设备，且有持续的流量数据刷新。

### Step 5（可选）: 模块打包与分发 `[x] 已完成`

- **Agent 任务**：编写或更新模块的 `update.json` 和 `customize.sh`，实现模块的在线更新支持。
- **具体动作**：
  1. 将 FlowCollect 二进制和 Clash 核心打包进同一个模块 zip。
  2. 在模块描述中注明集成状态。
- **已完成**：
  - `update.json`：`zipUrl` 和 `changelog` 指向 `0xav10086/box_for_magisk`
  - `module.prop`：`updateJson` 指向本项目 fork
  - `customize.sh`：音量键超时缩短为 5 秒，默认自动确认安装
  - 打包产物：`~/box_for_magisk/box_for_root-v1.10.2.zip` (2.7MB)
  - 包含内容：`flow_collect_client_android` + 完整模块结构

---

## 3. 桌面客户端架构细纲

### 前置准备

工作区：
- `~/FlowCollect`（FlowCollect 主仓库，含 `client/` 和 `server/`）
- `~/clash-verge-rev`（独立 clone 的 clash-verge-rev 仓库）

集成目标：在 Clash Verge Rev 的 Tauri 后端中，实现 Clash/Mihomo 核心 + FlowCollect 审计上报进程的 Sidecar 共存。当用户开启代理/启动核心时，静默拉起 FC 客户端；退出时一并销毁。

### Step 1: 扩展交叉编译 — Windows/macOS 桌面二进制 `[ ] 待执行`

- **Agent 任务**：在 `~/FlowCollect/client_build.sh` 中增加 Windows (amd64) 和 macOS (amd64/arm64) 的交叉编译目标。
- **具体动作**：
  1. 增加 `GOOS=windows GOARCH=amd64` 编译目标，产出 `flow_collect_client_windows_amd64.exe`。
  2. 增加 `GOOS=darwin GOARCH=amd64` 编译目标，产出 `flow_collect_client_darwin_amd64`。
  3. 增加 `GOOS=darwin GOARCH=arm64` 编译目标，产出 `flow_collect_client_darwin_arm64`。
  4. 所有产出物输出到 `~/FlowCollect/smart_spend/dist/`。
- **验收标准**：`file` 命令确认各二进制架构正确（Windows: PE32+ executable x86-64; macOS: Mach-O 64-bit executable x86_64/arm64）。

### Step 2: Rust 侧修改 — FlowCollect Sidecar 管理模块 `[ ] 待执行`

- **Agent 任务**：在 `~/clash-verge-rev/src-tauri/src/core/` 下新建 `flow_collect.rs`，实现 FC 客户端子进程的生命周期管理。
- **具体动作**：
  1. 新建 `flow_collect.rs`，导出 `start_flow_collect()` 和 `stop_flow_collect()` 函数。
  2. 使用 `std::process::Command` 启动 `flow_collect_client` 子进程，通过 `-c` 参数传递 Clash `config.yaml` 路径。
  3. 全局持有子进程句柄（`Mutex<Option<Child>>` 或 `OnceLock`），支持优雅停止（SIGTERM/TerminateProcess）+ 超时强杀。
  4. 二进制定位策略：从 Tauri sidecar 目录（`current_exe().parent()`）查找 `flow_collect_client` 可执行文件。
  5. 在 `mod.rs` 中导出新模块。
- **关键设计决策**：
  - 复用 Clash config.yaml 中的 `x-flow-collect` 扩展字段（与 Android 端一致），无需额外配置文件。
  - FC 客户端以普通用户权限运行（不需要 root/admin），因为它仅读取 Mihomo API 和发送 WebSocket。
- **验收标准**：`flow_collect.rs` 编译通过，函数签名正确。

### Step 3: Hook 到 CoreManager 生命周期 `[ ] 待执行`

- **Agent 任务**：修改 Clash Verge Rev 的核心生命周期管理代码，在启动/停止 Clash 核心时同步管理 FC 进程。
- **具体动作**：
  1. 修改 `src-tauri/src/core/manager/lifecycle.rs`：
     - `start_core()`：在 `start_core_by_sidecar()` 或 `start_core_by_service()` 成功后，调用 `flow_collect::start_flow_collect()`。
     - `stop_core()`：在停止核心之前，先调用 `flow_collect::stop_flow_collect()`。
  2. 修改 `src-tauri/src/feat/window.rs`：
     - `clean_async()`：在 `core_task` 中停止核心之前，先停止 FlowCollect 进程，确保退出时 FC 进程不残留。
- **验收标准**：启动 Clash 核心后，`ps | grep flow_collect` 可见进程；停止核心后进程消失。

### Step 4: Tauri 资源打包配置 `[ ] 待执行`

- **Agent 任务**：将 FlowCollect 客户端二进制注册为 Tauri sidecar 资源，确保打包后可被定位。
- **具体动作**：
  1. 方案 A（推荐）：在 `tauri.conf.json` 或 `Cargo.toml` 的 `[tauri.bundle]` 中注册 sidecar。
  2. 方案 B（轻量）：运行时从 `current_exe().parent()` 目录查找 `flow_collect_client`，无需 Tauri sidecar 机制。
  3. 在 GitHub Actions 或 Makefile.toml 中增加构建步骤，将 FlowCollect 二进制复制到 sidecar 目录。
- **验收标准**：打包后的 `.msi` / `.dmg` / `.AppImage` 中包含 `flow_collect_client` 二进制。

### Step 5: 桌面端端到端测试 `[ ] 待执行`

- **Agent 任务**：在真实的 Windows / macOS 机器上安装打包产物，验证数据链路。
- **具体动作**：
  1. 安装 Clash Verge Rev（含 FC 集成版）。
  2. 在 config.yaml 中配置 `x-flow-collect` 字段（remote-server、remote-token、device-id）。
  3. 开启系统代理 / 启动核心，确认 FC 客户端自动拉起。
  4. 访问服务端 `/api/stats`，确认该设备出现在设备列表中，且有持续的流量数据。
  5. 关闭核心 / 退出 Clash Verge Rev，确认 FC 进程被销毁。
- **验收标准**：服务端设备列表中出现桌面端设备，且退出后无残留进程。
