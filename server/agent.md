# FlowCollect 服务端子系统法典

> **Agent 读取指令**：接管服务端相关任务前，必须先阅读本文件。
> 本文档定义了 Go 服务端的架构目标、工作区目录规范、Gin 框架配置及安全策略。

## 1. 服务端架构总览

服务端以 **工作区目录** 形式部署，Go 后端承担以下核心职责：

1. **动态订阅分发枢纽**：读取 `templates/` 目录下的节点配置模板和 `RuleSet/` 规则集，通过 `/sub` 路由为客户端动态计算并下发 Clash 订阅配置。
2. **模板文件原始分发**：通过 `/templates/*filepath` 路由供 proxy-providers / rule-providers 拉取原始 YAML 文件。
3. **流量审计 API**：提供 REST API 和 WebSocket 端点，接收 Sidecar 上报的流量数据，存储至 SQLite，并向前端仪表盘实时推送。
4. **CF Tunnel 健康监控**：每 5 分钟检查 Cloudflare Tunnel 连通性，掉线时通过 Docker Engine API (Unix Socket) 自动重启容器。
5. **配置热更新**：通过 fsnotify 监听 INI 配置文件和 CSV 规则清单的变化，自动重载配置或重新编译规则集。

### 1.1 架构目标

| 目标 | 说明 | 状态 |
|------|------|------|
| 动态订阅分发 | 读取 `templates/*.yaml` + `templates/RuleSet/` + `templates/*.csv`，通过 `/sub` 路由生成个性化订阅链接 | `[x]` |
| 模板文件原始分发 | 通过 `/templates/*filepath` 供 proxy-providers / rule-providers 拉取 YAML 模板 | `[x]` |
| 流量审计 API | REST + WebSocket 接收 Sidecar 上报数据 | `[x]` |
| SQLite 持久化 | 流量数据本地存储（WAL 模式） | `[x]` |
| CORS + WebSocket 放行 | 允许跨域升级请求，放行 Cloudflare 特有头部 | `[x]` |
| TrustedPlatform 真实 IP | 适配 Cloudflare 隧道部署 | `[x]` |
| 动态配置覆盖 | `ReadMainSubConfig=true` 时，启动阶段从主订阅 YAML 提取 `mixed-port`/`port` 和 `secret`，覆盖 INI 的 `ListenPort`/`ServerToken` | `[x]` |
| CF Tunnel 健康监控 | 每 5 分钟检查外部 URL，失败时通过 Docker API 自动重启容器 | `[x]` |
| CSV 变更自动重编 | 监听 `86_rule_set_collect.csv` 变化，2 秒防抖后自动重新编译所有规则集 | `[x]` |
| INI 配置热更新 | 监听 `ServerSetting.ini` 变化，自动重载配置到内存 | `[x]` |
| 订阅启动时更新 | 服务启动后立即执行一次 `updateSubscriptionData()`，避免等待凌晨 cron | `[x]` |
| CSV 启动诊断 | 启动时输出 CSV 文件信息（大小、修改时间、有效规则数）和 RuleSet 文件列表 | `[x]` |

---

## 2. 工作区目录规范

服务端采用分层目录结构，将源码、配置、运行时数据和测试完全隔离：

```
server/
├── main.go                      # 入口：ensureDirs()、setupLogging()、Gin 引擎初始化、路由注册、cron 调度
├── config.go                    # 配置管理：加载/监听 ServerSetting.ini，支持热更新
├── docker.go                    # CF Tunnel 健康监控：通过 Docker Engine API (Unix Socket) 重启容器
├── handlers.go                  # REST API 处理：/api/auth、/api/stats、/api/devices、/report
├── sub_handler.go               # 订阅分发：读取 templates/ 动态生成 Clash 配置；模板文件原始分发
├── websocket.go                 # WebSocket 端点：/ws 实时流量上报接收与推送
├── service.go                   # 业务逻辑：订阅抓取、日报生成、邮件发送
├── yaml_config.go               # 模板与规则编译：下载订阅源、解析 CSV、编译 RuleSet；watchCSV 热更新；ExtractConfigFromMainSub
├── db.go                        # 数据库：SQLite 模型定义与初始化（WAL 模式）
├── fake_api.go                  # 仿真数据：/api/fake/stats 随机流量生成器
├── utils.go                     # 工具函数：流量格式化（formatNetworkBytes、formatBytes）
├── email_test.go                # 邮件告警测试（//go:build test）
├── go.mod / go.sum              # Go 依赖管理
│
├── configs/                     # 配置文件（*.ini 被 .gitignore 忽略）
│   ├── ServerSetting.ini        #   运行时配置（端口、Token、SMTP、订阅链接、CFTunnelContainer 等）
│   └── ServerSetting.ini.example#   配置模板（提交到仓库供参考）
│
├── templates/                   # 订阅模板与规则集（/sub 路由读取此目录）
│   ├── 86_rule_set_collect.csv  #   规则集清单（target → URL 映射，修改后自动重编规则集）
│   ├── *_nodes.yaml             #   节点模板文件（被 /sub 和 /templates/* 路由读取）
│   └── RuleSet/                 #   编译后的 Clash 规则集文件
│       ├── 86BemlyRules.yaml    #     bemly 策略组规则
│       ├── 86DirectRules.yaml   #     直连策略组规则
│       ├── 86HKRules.yaml       #     香港节点策略组规则
│       ├── 86JPRules.yaml       #     日本节点策略组规则
│       ├── 86USRules.yaml       #     美国节点策略组规则
│       ├── 86SwitchRules.yaml   #     切换策略组规则
│       └── 86RejectRules.yaml   #     拦截策略组规则
│
├── data/                        # 运行时数据（全部 Git 忽略，仅保留 .gitkeep）
│   ├── *.db                     #   SQLite 数据库文件
│   ├── *.db-shm / *.db-wal      #   WAL 模式共享内存文件
│   └── .gitkeep
│
├── logs/                        # 运行时日志（Git 忽略，仅保留 .gitkeep）
│   ├── server.log               #   服务端运行日志（同时输出到控制台和文件）
│   └── .gitkeep
│
├── test/                        # 测试脚本
│   └── test_server.sh           #   集成测试：自动读取 configs/ 中的端口，curl 验证 /sub + CORS
│
└── agent.md                     # 本文件：服务端子系统法典（AI Agent 必读）
```

**路径引用关系**：

| 代码中的常量/变量 | 实际指向 | 用途 |
|-------------------|----------|------|
| `iniPath` | `./configs/ServerSetting.ini` | 配置文件加载与热更新监听 |
| `TemplatesDir` | `./templates` | /sub 和 /templates/* 路由读取节点模板 |
| `RuleDir` | `./templates/RuleSet` | 规则集存放目录 |
| `CSVFile` | `./templates/86_rule_set_collect.csv` | 规则编译与 /sub 路由读取；watchCSV 热更新 |
| `conf.DBPath` | `./data/traffic.db`（默认值） | SQLite 数据库路径 |
| `conf.MainSubFile` | `./templates/{MainSubFile}` | `ReadMainSubConfig` 启用时，动态提取端口和 Token 的 YAML 源文件 |
| `conf.ListenPort` | `:7886`（默认值） | Gin HTTP 监听端口 |
| `conf.ServerToken` | `YourSecretToken`（默认值） | Bearer Token 鉴权密钥 |
| `conf.CFTunnelContainer` | 运行时配置 | Docker 容器名，用于 CF Tunnel 健康监控（空=禁用） |
| `conf.SubUrls` | INI 中的 SubUrls 段 | 订阅源映射（文件名→URL），`loadConfig()` 后立即生效 |
| 日志文件 | `./logs/server.log` | 运行日志输出（`setupLogging()` 同时输出到 stdout 和文件） |

**启动时自动创建**：`main.go` 中的 `ensureDirs()` 在任何业务逻辑之前自动创建 `data/`、`logs/`、`configs/`、`templates/` 目录，确保新环境部署不会因目录缺失而 panic。

### 2.1 部署模式

服务端支持 **Docker 镜像** 与 **二进制压缩包** 双轨分发：

| 模式 | 分发方式 | 资源注入方式 | 适用场景 |
|------|----------|-------------|----------|
| **Docker** | `ghcr.io/0xav10086/flow-collect-server:latest` | 全部通过 Volume 挂载（`configs/`、`templates/`、`data/`、`logs/`） | 云原生环境、CI/CD 流水线 |
| **二进制** | Release `.tgz` 压缩包 | 文件直接置于工作目录 | NAS、嵌入式设备、无 Docker 环境 |

**Docker 模式要点**：
- 镜像内仅包含编译好的二进制（`/app/flow_server`），不含任何源码、配置或数据文件。
- 四个运行时目录（`configs/`、`templates/`、`data/`、`logs/`）必须从宿主机 Volume 注入。
- 容器默认以 `appuser`（UID 1000）运行，宿主机挂载目录需确保可读写权限。
- CF Tunnel 健康检查需要额外挂载 Docker socket：`-v /var/run/docker.sock:/var/run/docker.sock`，且使用 `--user root`。
- 时区通过 `-e TZ=Asia/Shanghai` 环境变量注入（镜像已内置 `tzdata`）。

### 2.2 发布流程

服务端通过 GitHub Actions CI 构建并推送 Docker 镜像，**不需要本地 `go build`**。

**触发方式**：推送 tag `v*`（如 `v1.1.1`）到 GitHub 仓库，自动触发 CI。

**CI 流程**：
1. 编译服务端 Linux 二进制 + 构建前端静态资源 → 打包 `flow_collect.tgz`
2. 编译全平台 Sidecar 客户端二进制
3. 创建 GitHub Release 并上传所有构建产物
4. 构建 Docker 镜像并推送到 GHCR

**镜像地址**：`ghcr.io/0xav10086/flow-collect-server:{version}`

**等待时间**：推送 tag 后，sleep 140 秒等待 CI 完成。

**NAS 部署命令**（单行，不要使用 `\` 换行）：
```bash
sudo docker pull ghcr.io/0xav10086/flow-collect-server:v1.1.3 && sudo docker stop flow-collect && sudo docker rm flow-collect && sudo docker run -d --name flow-collect --user root --restart unless-stopped -v /var/run/docker.sock:/var/run/docker.sock -v /volume1/docker/flow_collect/configs:/app/configs -v /volume1/docker/flow_collect/templates:/app/templates -v /volume1/docker/flow_collect/data:/app/data -v /volume1/docker/flow_collect/logs:/app/logs -p 7886:7886 -e TZ=Asia/Shanghai ghcr.io/0xav10086/flow-collect-server:v1.1.3
```

注意：
- `{version}` 替换为实际的 tag 名（如 `v1.1.1`）
- `--user root` + `-v /var/run/docker.sock:/var/run/docker.sock` 用于 CF Tunnel 容器重启
- NAS 路径 `/volume1/docker/flow_collect/` 下需提前准备好 `configs/`、`templates/`、`data/`、`logs/` 目录

---

## 3. 核心路由职责

### 3.1 订阅分发路由（`/sub`）

读取 `templates/` 目录下的节点模板和规则集，根据客户端请求参数动态拼接完整的 Clash 配置文件，通过 HTTP 响应返回。

```
客户端 GET /sub?device=xxx?token=YourSecretToken
  → Go 后端读取 templates/*.yaml + templates/RuleSet/ + templates/86_rule_set_collect.csv
  → 动态计算并拼接完整 Clash 配置（proxy-providers、rule-providers、rules 等）
  → 返回 text/yaml 响应
```

鉴权：URL 参数 `token` 验证（`queryTokenAuth()`），**不是** Bearer Header。

### 3.2 模板文件原始分发（`/templates/*filepath`）

供 Clash 内核的 proxy-providers / rule-providers 直接拉取原始 YAML 文件。

```
客户端 GET /templates/PowerBy0xav10086?token=YourSecretToken
  → 返回 ./templates/PowerBy0xav10086 文件的原始内容
  → 用于 proxy-providers: { path: "...", url: "https://nas.0xav10086.space/templates/..." }
```

鉴权：URL 参数 `token` 验证，与 `/sub` 共享鉴权逻辑。

### 3.3 流量审计 API（`/api/*`）

| 路由 | 方法 | 鉴权 | 说明 |
|------|------|------|------|
| `/api/auth` | POST | 无 | 设备认证 |
| `/api/stats` | GET | Bearer Token | 获取流量统计 |
| `/api/devices` | GET | Bearer Token | 获取设备列表 |
| `/api/fake/stats` | GET | Bearer Token | 随机仿真流量数据 |
| `/api/trigger-update` | POST | Bearer Token | 手动触发订阅与规则更新 |

### 3.4 流量上报与 WebSocket

| 路由 | 方法 | 鉴权 | 说明 |
|------|------|------|------|
| `/report` | POST | Bearer Token | Sidecar 上报流量数据 |
| `/ws` | GET | URL Token 参数 | WebSocket 实时流量推送 |

### 3.5 Token 鉴权方式

代码中存在 **两种鉴权方式**，不可混用：

| 方式 | 适用路由 | 验证方式 |
|------|----------|----------|
| **Bearer Header** | `/api/*`、`/report` | `Authorization: Bearer YourSecretToken`（`TokenAuthMiddleware()`）|
| **URL Query** | `/sub`、`/ws`、`/templates/*` | `?token=YourSecretToken`（`queryTokenAuth()`）|

---

## 4. Gin 框架配置

### 4.1 CORS 中间件

在 `main.go` 中以 Gin 中间件形式实现，配置如下：
- 允许所有来源（`Access-Control-Allow-Origin: *`）
- 允许凭据（`Access-Control-Allow-Credentials: true`）
- 允许头部：`Content-Type`, `Authorization`, `Upgrade`, `Sec-WebSocket-*` 系列, `CF-Connecting-IP`, `CF-IPCountry`, `CF-Ray` 等
- 允许方法：`POST, OPTIONS, GET, PUT, DELETE, PATCH`
- OPTIONS 预检请求直接返回 204

### 4.2 TrustedPlatform 获取真实 IP

```go
r.TrustedPlatform = gin.PlatformCloudflare
```

当服务端部署在 Cloudflare 隧道之后时，从 `CF-Connecting-IP` 请求头获取客户端真实 IP。

---

## 5. 热更新行为

### 5.1 INI 配置热更新（`watchConfig()` in `config.go`）

触发条件：`configs/ServerSetting.ini` 文件写入事件（`fsnotify.Write`）。

行为：
- 调用 `loadConfig()` 重新解析整个 INI 文件，更新 `conf` 结构体
- **SubUrls** — 内存中的订阅 URL map 立即更新，但不会自动触发订阅下载（需要等 cron 或手动 `/api/trigger-update`）
- **CFTunnelContainer** — 下次健康检查（每 5 分钟）自动读取新值
- **ServerToken / ListenPort** — `loadConfig()` 更新内存值，但已注册的 Gin 路由不会重新绑定

### 5.2 CSV 变更自动重编（`watchCSV()` in `yaml_config.go`）

触发条件：`templates/86_rule_set_collect.csv` 文件的写入或创建事件。

行为：
1. 2 秒防抖（防止编辑器多次写入触发重复编译）
2. 调用 `processRules()`：
   - 截断所有 `86*Rules.yaml` 文件至 `[MANUAL_END] Private` 标记处
   - 重新读取 CSV 文件
   - 对每条有效规则记录，下载远程规则文件并追加到对应的策略组文件
3. 新内容对下一次 `/sub` 请求立即可见

### 5.3 CF Tunnel 健康监控（`docker.go`）

| 项目 | 说明 |
|------|------|
| 检查周期 | 每 5 分钟（cron: `*/5 * * * *`）|
| 检查方式 | HTTP GET `https://nas.0xav10086.space/`，超时 5s |
| 失败条件 | 网络错误 或 HTTP 状态码 >= 500 |
| 失败动作 | 立即调用 `dockerRestartContainer(container)` |
| 重启方式 | Docker Engine API via Unix Socket (`/var/run/docker.sock`) |

容器名从 `ServerSetting.ini` 的 `CFTunnelContainer` 字段读取。

---

## 6. 启动流程

```
main()
├── ensureDirs()                    # 创建 data/ logs/ configs/ templates/
├── setupLogging()                  # 初始化日志（stdout + ./logs/server.log）
├── loadConfig()                    # 加载 ServerSetting.ini
├── ExtractConfigFromMainSub()      # 如果 ReadMainSubConfig=true，覆盖端口和 Token
├── logCSVDiagnostics()             # 输出 CSV 和 RuleSet 文件诊断信息
├── go watchConfig()                # 启动 INI 文件监听
├── go watchCSV()                   # 启动 CSV 文件监听
├── initDB()                        # 初始化 SQLite（WAL 模式）
├── cron.Start()                    # 注册定时任务（日报/清理/订阅更新/CF Tunnel 检查）
├── CFTunnelHealthCheck()           # 启动时立即执行一次隧道检查
├── updateSubscriptionData()        # 启动时立即更新一次订阅数据
└── r.Run(port)                     # 启动 Gin HTTP 服务
```
