# FlowCollect 全局架构宪法 (Agent 核心知识库)

> **Agent 读取指令**：
> 本文档定义了 FlowCollect 的宏观演进目标、架构规范和核心禁忌。
> **接管具体子系统任务前，必须先读取对应的子系统法典（见 §4 路由表）。**
> **严格禁止**试图一次性完成所有任务。必须按照人类分配的单一子任务进行局部迭代。

> **项目进度**：当前项目的整体开发进度与待办事项，请查阅 `ROADMAP.md`。

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
| **服务端**（Go + Gin） | `server/agent.md` | 工作区目录部署、Gin 配置、CORS WebSocket 放行、TrustedPlatform 真实 IP、动态配置覆盖（ReadMainSubConfig） |
| **前端**（Vue 3 SPA） | `smart_spend/agent.md` | metacubexd iframe 嵌入、环境变量规范、安全约束 |

---

## 5. AI 协作与 Git 代码提交流程

Agent 在完成任何代码修改任务后，若人类要求进行版本控制提交，必须严格遵守以下规范：

1. **原子化提交**：只提交当前子系统任务相关的代码变更。
2. **AI 身份隔离**：执行 `git commit` 时，必须强制携带身份伪装参数，严禁使用宿主机默认 Git 用户提交。
   - 示例命令：`git commit --author="Claude AI <claude@anthropic.com>" -m "feat(scope): description"`
3. **Commit Message 规范**：严格遵循 Angular 规范（如 `feat`, `fix`, `refactor`, `docs`），并在括号内标明影响的子模块（如 `client`, `server`, `smart_spend`）。
