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



## 2. 系统架构

* **客户端 (Client)**：部署于 Windows (Clash Verge Rev)、Android (KernelSU + Box) 和 OpenWrt (Nikki)。每 10 秒请求本地内核 API，计算流量增量 (Delta) 并推送。
* **服务端 (Server)**：部署于 Ubuntu VPS。采用 Gin 框架处理请求，GORM + 纯 Go 版 SQLite (`glebarez/sqlite`) 实现无 CGO 依赖的数据持久化。

## 3. 快速开始

### 3.1 服务端部署 (VPS)

1. **编译** (在 Windows 执行以规避 VPS 性能不足问题)：
```powershell
$env:GOOS="linux"; $env:GOARCH="amd64"; go build -o server server.go

```


2. **上传并运行**：
```bash
scp ./server user@vps_ip:~/traffic-server/
./server

```



### 3.2 客户端部署

1. **配置**：修改 `client.go` 中的 `RemoteServer` 为 `http://vps_ip:8686/report`。
2. **编译** (以 OpenWrt 为例)：
```powershell
$env:GOOS="linux"; $env:GOARCH="mipsle"; go build -o tracker_router client.go

```



## 4. 配置说明

| 配置项 | 说明 | 默认值 |
| --- | --- | --- |
| `MihomoAPIAddr` | 本地内核外部控制地址 | `127.0.0.1:9097` |
| `MihomoSecret` | 内核 API 访问密钥 | `abcd` |
| `ListenPort` | 服务端 API 监听端口 | `8686` |
| `Interval` | 采样上报周期 | `10s` |

## 5. 待办事项 (Roadmap)

* [ ] 开发基于 Vue3 的可视化 Web 前端面板。
* [ ] 增加多用户隔离统计功能。
* [ ] 支持 Telegram Bot 实时流量查询。

---

**下一步建议**：
既然 README 已经写好，你可以通过 `git init` 初始化本地仓库，并使用 `git remote add origin [你的私有仓库链接]` 将代码推送到 GitHub。

**需要我为你编写一个 `.gitignore` 文件，以防止你的 `traffic.db` 数据库和 QQ 邮箱授权码等敏感信息被误上传吗？**