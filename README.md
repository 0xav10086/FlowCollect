# FlowCollect: 分布式流量审计系统

**FlowCollect** 是一个专为代理环境设计的轻量级分布式流量审计系统。
它的主要目标是统计多端代理流量，分析机场数据，并发送每日审计日报。

## 1. 项目概览与架构

* **目标**：统计多端代理流量，分析机场数据，发送每日审计日报。
* **架构**：分布式 Client (Go) -> 中央 Server (Go/Gin) -> 可视化 Dashboard (Vue3)。

### 📁 项目结构

```text
FlowCollect/
├── client/          # Go 客户端 (多平台交叉编译)
├── server/          # Go 服务端 (数据中心与定时任务)
├── smart_spend/     # Vue 3 可视化控制面板 (SmartSpend)
├── client_build.sh   # 客户端一键编译脚本 (Bash)
├── server_build.sh  # 服务端与前端一键编译脚本 (Bash)
└── go.work          # Go Workspace 工作区配置
```

## 2. 技术栈

* **后端 (Server/Client)**: Go 1.18+, Gin, GORM, SQLite (`glebarez/sqlite` 纯 Go 驱动), cron, fsnotify, ini.v1。
* **前端 (SmartSpend)**: Vue 3, TypeScript, Vite, Tailwind CSS, Element Plus, Anime.js, ECharts。
* **配置管理**: 使用 `.ini` 文件进行热加载配置。

## 3. 核心特性

* **多端适配**：基于 Go 语言开发，支持交叉编译至 Windows、Linux 及 OpenWrt 等多平台。
* **精细化统计**：
  * 基于全局流量监控，自动识别并区分代理流量（Proxy）与直连流量（Direct）。
  * 智能数据处理：启动时静默初始化快照，防止首包数值爆表。
* **云端自动化审计**：
  * **机场联动**：解析订阅 Header (`Subscription-Userinfo`) 获取机场剩余流量。
  * **余量预测与泄露预警**：估算可用天数，对比机场记录与本地统计检测代理泄露风险。
  * **每日日报**：每晚 23:55 通过 QQ 邮箱发送详细流量分析报表。

## 4. 开发规范与约束

* **模块化**: 采用 Go Workspace (`go.work`) 统一管理 `client` 和 `server` 目录。
* **路由规则**: 后端 API 统一前缀为 `/api`；流量上报接口为 `POST /report`。
* **安全性与隐私规范**: 
  * 严禁在代码中硬编码敏感信息；所有网络请求或邮件发送的配置项应抽象到 `Config` 结构体中，从 `ClientSetting.ini` 或 `ServerSetting.ini` 读取。
  * **双仓库推送机制**：
    * **公开仓库**：严格禁止出现任何 `.ini` 文件或敏感 Token。
    * **私密仓库**：作为全量代码备份。

## 5. 快速开始与部署

### 5.1 服务端与面板部署 (VPS)

1. **一键编译** (执行新编写的脚本，自动编译前端和后端可执行文件)：
   ```bash
   ./server_build.sh
   ```
2. 编译成功后，在 `smart_spend/dist/` 中会生成 `flow_collect_server` 可执行文件。
3. **上传并运行**：将该二进制文件及 `dist` 中的前端静态资源一同部署至 VPS，并在同目录配置 `ServerSetting.ini` 后运行。

### 5.2 客户端部署

1. **配置**：根据模板配置 `ClientSetting.ini`。
2. **编译**：使用 `client_build.sh` 脚本进行一键跨平台分发与编译，脚本会自动通过 `-ldflags` 注入设备相关的变量。

## 6. 当前进度 (Roadmap)

* [x] 完成带热更新功能的 Go 客户端与服务端。
* [x] 完成跨平台自动化编译脚本 (`client_build.sh`、`server_build.sh`)。
* [ ] **正在进行**：开发 SmartSpend (Vue3) 可视化看板，对接 `GET /api/stats`。
* [ ] 增加多用户隔离统计功能。
* [ ] 支持 Telegram Bot 实时流量查询。
