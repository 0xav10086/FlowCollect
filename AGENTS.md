# FlowCollect 项目上下文 (AI 读取专用)

## 1. 项目概览
本项目名为 FlowCollect，是一个分布式流量审计系统。
- **目标**：统计多端代理流量，分析机场数据，发送每日审计日报。
- **架构**：分布式 Client (Go) -> 中央 Server (Go/Gin) -> 可视化 Dashboard (Vue3)。

## 2. 技术栈
- **后端 (Server/Client)**: Go 1.18+, Gin, GORM, SQLite (glebarez/sqlite), cron, fsnotify, ini.v1。
- **前端 (SmartSpend)**: Vue 3, TypeScript, Vite, Tailwind CSS, Element Plus, Anime.js, ECharts。
- **配置管理**: 使用 .ini 文件进行热加载配置。

## 3. 开发规范与约束
- **模块化**: 采用 Go Workspace (`go.work`) 管理 `client` 和 `server` 目录。
- **参数注入**: 客户端变量（如 DeviceID）必须定义为 `var`，以便通过 `distribute.ps1` 中的 `-ldflags` 进行编译时注入。
- **安全性**: 严禁在代码中硬编码敏感信息；所有配置项应从 `ClientSetting.ini` 或 `ServerSetting.ini` 读取。
- **路由规则**: 后端 API 统一前缀为 `/api`；流量上报接口为 `POST /report`。

## 4. 当前进度 (Roadmap)
- [x] 完成带热更新功能的 Go 客户端与服务端。
- [x] 完成跨平台自动化编译脚本。
- [ ] 正在进行：开发 SmartSpend (Vue3) 可视化看板，对接 `GET /api/stats`。

## 
### 🔒 隐私安全规范
- 本项目采用双仓库推送机制。
- **公开仓库**：严格禁止出现任何 .ini 文件或敏感 Token。
- **私密仓库**：作为全量备份。
- 在编写涉及网络请求或邮件发送的代码时，务必将配置项抽象到 `Config` 结构体中，并通过读取 `.ini` 文件初始化，严禁硬编码。