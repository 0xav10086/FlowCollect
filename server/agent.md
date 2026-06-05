# FlowCollect 服务端子系统法典

> **Agent 读取指令**：接管服务端相关任务前，必须先阅读本文件。
> 本文档定义了 Go 服务端的架构目标、工作区目录规范、Gin 框架配置及安全策略。

## 1. 服务端架构总览

服务端以 **工作区目录** 形式部署，Go 后端承担两大核心职责：

1. **动态订阅分发枢纽**：读取 `templates/` 目录下的 `*.yaml` 节点配置模板和 `RuleSet/` 规则集，通过 HTTP 路由（如 `/sub`）为客户端动态计算并下发 Clash 订阅配置。
2. **流量审计 API**：提供 REST API 和 WebSocket 端点，接收 Sidecar 上报的流量数据，存储至 SQLite，并向前端仪表盘实时推送。

后台脚本（如 `auto_update_node_and_rule.sh`）可独立维护 `templates/RuleSet/` 和 `templates/*.yaml` 配置，Go 后端实时读取最新状态，无需重启或重新编译。

### 1.1 架构目标

| 目标 | 说明 | 状态 |
|------|------|------|
| 动态订阅分发 | 读取 `templates/*.yaml` + `templates/RuleSet/` + `templates/*.csv`，通过 `/sub` 路由生成个性化订阅链接 | `[x]` |
| 流量审计 API | REST + WebSocket 接收 Sidecar 上报数据 | `[x]` |
| SQLite 持久化 | 流量数据本地存储 | `[x]` |
| CORS + WebSocket 放行 | 允许跨域升级请求 | `[x]` |
| TrustedPlatform 真实 IP | 适配 Cloudflare 隧道部署 | `[x]` |
| 动态配置覆盖 | `ReadMainSubConfig=true` 时，启动阶段从主订阅 YAML 提取 `mixed-port`/`port` 和 `secret`，覆盖 INI 的 `ListenPort`/`ServerToken` | `[x]` |

---

## 2. 工作区目录规范

服务端采用分层目录结构，将源码、配置、运行时数据和测试完全隔离：

```
server/
├── main.go                      # 入口：ensureDirs()、Gin 引擎初始化、路由注册
├── config.go                    # 配置管理：加载/监听 configs/ServerSetting.ini
├── handlers.go                  # REST API 处理：/api/auth、/api/stats、/report
├── sub_handler.go               # 订阅分发：读取 templates/ 动态生成 Clash 配置
├── websocket.go                 # WebSocket 端点：/ws 实时流量上报接收
├── service.go                   # 业务逻辑：订阅抓取、日报生成、邮件发送
├── yaml_config.go               # 模板与规则编译：下载订阅源、解析 CSV、编译 RuleSet；ExtractConfigFromMainSub 动态配置提取
├── db.go                        # 数据库：SQLite 模型定义与初始化（WAL 模式）
├── fake_api.go                  # 仿真数据：/api/fake/stats 随机流量生成器
├── utils.go                     # 工具函数：流量格式化（formatNetworkBytes 等）
├── email_test.go                # 邮件告警测试（//go:build test）
├── go.mod / go.sum              # Go 依赖管理
│
├── configs/                     # 配置文件（*.ini 被 .gitignore 忽略）
│   ├── ServerSetting.ini        #   运行时配置（端口、Token、SMTP、订阅链接）
│   └── ServerSetting.ini.example#   配置模板（提交到仓库供参考）
│
├── templates/                   # 订阅模板与规则集（/sub 路由读取此目录）
│   ├── 86_rule_set_collect.csv  #   规则集清单（target → URL 映射）
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
│   ├── server.log               #   服务端运行日志
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
| `TemplatesDir` | `./templates` | /sub 路由读取节点模板 |
| `RuleDir` | `./templates/RuleSet` | /sub 路由读取规则集 |
| `CSVFile` | `./templates/86_rule_set_collect.csv` | 规则编译与 /sub 路由读取 |
| `conf.DBPath` | `./data/traffic.db`（默认值） | SQLite 数据库路径 |
| `conf.MainSubFile` | `./templates/{MainSubFile}` | `ReadMainSubConfig` 启用时，动态提取端口和 Token 的 YAML 源文件 |
| 日志文件 | `./logs/server.log` | 运行日志输出 |

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
- 时区通过 `-e TZ=Asia/Shanghai` 环境变量注入（镜像已内置 `tzdata`）。

### 2.2 发布流程

服务端通过 GitHub Actions CI 构建并推送 Docker 镜像，**不需要本地 `go build`**。

**触发方式**：推送 tag `v*`（如 `v1.1.0`）到 GitHub 仓库，自动触发 CI。

**CI 流程**：
1. 编译服务端 Linux 二进制 + 构建前端静态资源 → 打包 `flow_collect.tgz`
2. 编译全平台 Sidecar 客户端二进制
3. 创建 GitHub Release 并上传所有构建产物
4. 构建 Docker 镜像并推送到 GHCR

**镜像地址**：`ghcr.io/0xav10086/flow-collect-server:{version}`

**等待时间**：推送 tag 后，sleep 140 秒等待 CI 完成。

**NAS 部署命令**（单行，不要使用 `\` 换行）：
```bash
docker pull ghcr.io/0xav10086/flow-collect-server:v1.1.0 && docker stop flow_server && docker rm flow_server && docker run -d --name flow_server --user root --restart unless-stopped -v /var/run/docker.sock:/var/run/docker.sock -v /volume1/docker/flow_collect/configs:/app/configs -v /volume1/docker/flow_collect/templates:/app/templates -v /volume1/docker/flow_collect/data:/app/data -v /volume1/docker/flow_collect/logs:/app/logs -p 7886:7886 -e TZ=Asia/Shanghai ghcr.io/0xav10086/flow-collect-server:v1.1.0
```

注意：
- `{version}` 替换为实际的 tag 名（如 `v1.1.0`）
- `--user root` + `-v /var/run/docker.sock:/var/run/docker.sock` 用于 CF Tunnel 容器重启
- NAS 路径 `/volume1/docker/flow_collect/` 下需提前准备好 `configs/`、`templates/`、`data/`、`logs/` 目录

---

## 3. 核心路由职责

### 3.1 订阅分发路由（`/sub`）

Go 后端读取 `templates/` 目录下的节点模板和规则集，根据客户端请求参数（如设备 ID）动态拼接完整的 Clash 配置文件，通过 HTTP 响应返回。

```
客户端 GET /sub?device=xxx
  → Go 后端读取 templates/*.yaml + templates/RuleSet/ + templates/86_rule_set_collect.csv
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
