# FlowCollect: 分布式流量审计系统

**FlowCollect** 是一个专为代理环境设计的轻量级分布式流量监控方案。它通过主动轮询各端（Windows, Android, OpenWrt）的代理内核 API，实现对节点级流量的精确统计，并在云端 VPS 进行数据汇总与每日审计分析。

## 1. 核心特性

* **多端适配**：基于 Go 语言开发，支持交叉编译至 Windows (amd64)、Linux (amd64/arm64) 及 OpenWrt (mipsle/arm) 等多平台。
* **精细化统计**：
* 基于 `utun` 设备的全局流量监控。
* 自动识别并区分代理流量（Proxy）与直连流量（Direct）。
* 支持特定本地节点（如 `ua3f`）的特殊标记处理。


* **智能数据处理**：
* **静默初始化**：启动时自动抓取连接快照，防止首包因历史统计导致的数值爆表。
* **单位自动换算**：控制台实时显示易读的 B/KB/MB/GB 单位。


* **云端自动化审计**：
* **机场联动**：通过解析订阅 Header (`Subscription-Userinfo`) 实时获取机场剩余流量。
* **余量预测**：基于每日消耗自动估算机场订阅的剩余可用天数。
* **泄露预警**：对比机场记录与本地统计，自动检测是否存在代理泄露风险。
* **每日日报**：每晚 23:55 通过 QQ 邮箱发送详细的流量分析报表。

### 🏗️ 核心架构

* **客户端 (Client)**：部署于 Windows, Android 或 OpenWrt。每 10 秒计算流量增量并通过热加载的 `.ini` 配置文件管理参数。
* **服务端 (Server)**：部署于 VPS。使用 Gin 提供 API，GORM + SQLite 持久化数据，并集成每日邮件报表。
* **控制面板 (SmartSpend)**：基于 Vue 3 的现代化前端，直观展示各端流量消耗与审计结果。

### 📁 项目结构

```text
FlowCollect/
├── client/          # Go 客户端 (多平台交叉编译)
├── server/          # Go 服务端 (数据中心与定时任务)
├── smart_spend/     # Vue 3 可视化控制面板
├── distribute.ps1   # 一键分发编译脚本 (PowerShell)
└── go.work          # Go Workspace 工作区配置
```

## 2. 快速开始

### 2.1 服务端部署 (VPS)

1. **编译** (在 Windows 执行以规避 VPS 性能不足问题)：
```powershell
$env:GOOS="linux"; $env:GOARCH="amd64"; go build -o server server.go
```


2. **上传并运行**：
```bash
scp ./server user@vps_ip:~/traffic-server/
./server
```



### 2.2 客户端部署

1. **配置**：修改 `client.go` 中的 `RemoteServer` 为 `http://vps_ip:1234/report`。
2. **编译** (以 OpenWrt 为例)：
```powershell
$env:GOOS="linux"; $env:GOARCH="mipsle"; go build -o tracker_router client.go
```



## 3. 配置说明

本系统使用 `.ini` 文件进行配置。出于安全考虑，真实的配置文件已被 `git ignore` 忽略。

### 3.1 获取配置模板
1. 在 `client/` 目录下创建 `ClientSetting.ini`。
2. 在 `server/` 目录下创建 `ServerSetting.ini`。

您可以参考项目中的 `setting.ini.example` 进行配置。

### 3.2 关键参数说明
| 参数名 | 说明 |
| :--- | :--- |
| `MihomoSecret` | 您 Clash/Mihomo 内核设置的 API 密码 |
| `RemoteToken` | 客户端与服务端通信的鉴权令牌（两端需保持一致） |
| `EmailPass` | 发送日报的邮箱授权码（建议使用专用的 SMTP 授权码） |

## 4. 待办事项 (Roadmap)

* [ ] 开发基于 Vue3 的可视化 Web 前端面板。
* [ ] 增加多用户隔离统计功能。
* [ ] 支持 Telegram Bot 实时流量查询。

---
