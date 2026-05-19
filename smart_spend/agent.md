# FlowCollect 前端子系统法典

> **Agent 读取指令**：接管前端相关任务前，必须先阅读本文件。
> 本文档定义了 Vue 3 前端工程（smart_spend）的架构规范、metacubexd 集成细节及安全约束。

## 1. 前端架构总览

`smart_spend` 是纯粹的 Vue 3 前端工程。其 `dist/` 目录只允许存放 HTML/CSS/JS 等前端静态资源，绝不允许污染任何跨平台二进制可执行文件。

### 1.1 前端整合：MetaCubeX/metacubexd `[x] 已完成`

- **目标**：将 metacubexd 的能力融入 FlowCollect 现有的 Vue 3 仪表盘中。
- **集成位置**：`EquipmentStatus.vue`。
- **工作流**：用户在设备详情页点击时，能唤起 metacubexd 面板，静默传递 `hostname`、`port` 和 `secret` 进行连接，避免视觉割裂感。

---

## 2. metacubexd 集成实现细节

### 2.1 iframe 静默嵌入

已实现通过 iframe 及 URL Query（`hostname` / `port` / `secret`）完成 metacubexd 在 `EquipmentStatus.vue` 中的无状态静默直连嵌入。

### 2.2 涉及的改造点

| 模块 | 改造内容 |
|------|----------|
| `EquipmentStatus.vue` | 嵌入 metacubexd iframe，拼接 URL Query 参数 |
| `Sidebar.vue` / `MobileBottomNav.vue` | UI 裁剪，移除不需要的导航入口 |
| `auth.global.ts` | 路由守卫改造，允许 metacubexd 面板页面免登录访问 |
| `endpoint.ts` | 临时端点机制，支持动态切换后端地址 |
| Nuxt SPA 构建 | 确保 SSG/SSR 不影响 iframe 嵌入的正确性 |

### 2.3 Gin StaticFS 托管

前端构建产物通过服务端 Gin 的 `StaticFS` 托管，metacubexd 面板产物位于 `web/node-panel/` 子目录。

---

## 3. 环境变量与安全规范

### 3.1 环境变量

| 变量名 | 用途 | 示例 |
|--------|------|------|
| `VITE_API_BASE_URL` | 后端 API 基础地址 | `https://your-domain.com/api` |

所有环境变量通过 Vite 的 `import.meta.env` 机制注入，禁止在源码中硬编码任何 URL 或密钥。

### 3.2 安全约束

- 前端**绝不**存储或暴露代理节点的明文信息（如 `secret`、`token`）。
- iframe 传递的 `secret` 参数仅在 URL Query 中短暂出现，前端不得将其持久化到 `localStorage` 或 `sessionStorage`。
- 所有 API 请求必须通过 HTTPS 或 Cloudflare 隧道传输，禁止明文 HTTP。
