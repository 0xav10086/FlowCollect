# FlowCollect 下一代架构上下文 (Agent 核心知识库)

> **Agent 读取指令**：
> 本文档定义了 FlowCollect 的宏观演进目标和架构规范。在任何对话中，你（Agent）必须以此为基准。
> **严格禁止**试图一次性完成所有任务。必须按照人类分配的单一子任务（如“仅修改前端 UI”或“仅编译 Android 模块”）进行局部迭代。

## 1. 终极愿景：主动式控制与数据平面
FlowCollect 正在从一个被动的分布式流量审计系统升级为深度集成的代理与审计一体化平台。
核心理念：**客户端即代理节点，服务端即单文件部署面板。**

## 2. 三大开源项目的 Fork 与集成职责

### 2.1 前端整合：MetaCubeX/metacubexd
- **目标**：将其能力融入 FlowCollect 现有的 Vue 3 仪表盘中。
- **集成位置**：`EquipmentStatus.vue`。
- **工作流**：用户在设备详情页点击时，能唤起 metacubexd 面板，静默传递 `hostname`、`port` 和 `secret` 进行连接，避免视觉割裂感。

### 2.2 Android 客户端架构：taamarin/box_for_magisk
- **目标**：实现底层的透明代理与无感上报。
- **集成思路**：在 Magisk/KernelSU 模块的启动脚本中，拉起 Clash Meta 代理内核的同时，拉起 FlowCollect 的审计上报进程（Go 编译的二进制文件）。两者需实现进程级共存。

### 2.3 桌面客户端架构：clash-verge-rev/clash-verge-rev
- **目标**：构建“既能代理又能上报”的融合型可执行文件。
- **集成思路**：利用 Go 语言的特性，将 mihomo (Clash Meta) 作为 Library 引入 FlowCollect 的 Client 代码中。使用 Goroutine 并发运行代理逻辑和审计上报逻辑，生成单一二进制文件 `fc-core`，再由 verge-rev 的前端 GUI 进行控制。

### 2.4 服务端架构：极简单文件部署
- **目标**：摒弃以前将 `dist` 编译文件和 Go 文件放在一起运行的繁琐模式[cite: 3]。
- **集成思路**：使用 Go 的 `//go:embed` 特性，在编译期将 Vue 3 的前端产物直接打包进 Go 二进制文件中，实现真正的单文件运行。

## 3. Agent 执行与开发约束
- **模块化边界**：严格维护 `client` 和 `server` 目录的隔离[cite: 3]。
- **安全与配置**：禁止硬编码。所有配置通过 `.ini` 读取[cite: 3]，网络请求参数通过结构体抽象。
- **状态确认机制**：Agent 在修改任何关键文件后，必须输出测试方案，等待人类反馈“测试通过”后，才能推进到下一步。

## 4.前端整合细纲

### 阶段一：定制与魔改 `MetaCubeX/metacubexd` (独立项目操作)

在这个阶段，Agent 的工作区是单独 clone 下来的 `metacubexd` 仓库。

**Step 1: 屏蔽“配置”选项卡 (UI 裁剪)**
*   **Agent 任务**：在 `metacubexd` 的源码中（通常在路由配置文件或侧边栏组件，如 `src/components/Sidebar` 或 `src/router` 相关文件），找到渲染侧边栏菜单的列表。
*   **具体动作**：定位到名称为“配置”或路径为 `/settings` (或类似路径) 的菜单项，将其注释掉或通过条件渲染（如设置一个不显示的 flag）隐藏。
*   **验收标准**：本地运行 `pnpm dev` 时，侧边栏不再显示“配置”图标，但其他如“概览”、“代理”、“连接”等均正常工作。

**Step 2: 注入“静默登录”逻辑 (核心改造)**
*   **Agent 任务**：`metacubexd` 默认打开时如果未配置后端，会弹出一个登录/连接页面。我们需要让它支持通过 URL 参数一键直连。
*   **具体动作**：修改应用的初始化逻辑（通常在 `src/store`、`src/api` 或入口文件 `App.jsx`/`App.vue` 中），使其优先读取 URL 中的 `Query Parameters`（例如：`?hostname=192.168.1.100&port=9090&secret=your_token`）。
*   **逻辑要求**：如果检测到这三个参数，直接将它们写入应用的本地状态（LocalStorage/State），跳过手动输入的连接界面，直接建立 WebSocket/HTTP 连接。

**Step 3: 编译与打包产物**
*   **Agent 任务**：执行构建命令（`pnpm build`）。
*   **产出物**：获取 `dist` 目录下的所有静态文件。这就是我们真正需要的“定制版节点监视器”。

---

### 阶段二：集成到 `FlowCollect` (回到主项目)

在这个阶段，Agent 的工作区回到你的 FlowCollect 项目（Vue 3 + Go）。

**Step 4: 静态资源托管 (Go 服务端)**
*   **Agent 任务**：将阶段一得到的定制版 `metacubexd/dist` 文件夹复制到 FlowCollect 的某个目录下（例如 `smart_spend/dist/metacubexd/`）。
*   **Go 路由配置**：如果不使用 `//go:embed`，则在 Gin 路由中增加静态目录映射：
    ```go
    // 假设访问 /node-panel/ 就会加载我们的定制版 metacubexd
    r.StaticFS("/node-panel", http.Dir("./smart_spend/dist/metacubexd"))
    ```

**Step 5: 修改 `EquipmentStatus.vue` (前端嵌入)**
*   **Agent 任务**：在设备的详情页面，增加一个 UI 入口并利用 `iframe` 嵌入定制面板。
*   **具体动作 1 (增加入口)**：在界面上添加一个按钮，例如“⚡ 实时节点状态”。
*   **具体动作 2 (编写抽屉/模态框组件)**：使用 Element Plus 的 `el-drawer` 或 `el-dialog` 组件，设置宽度为 `80%` 或全屏。
*   **具体动作 3 (动态 iframe 挂载)**：在抽屉内部放入 `<iframe>`。当用户点击某台设备时，获取该设备的 IP、API 端口和 Token。
*   **代码生成要求 (Agent 需输出类似以下代码)**：
    ```vue
    <template>
      <el-drawer v-model="drawerVisible" title="节点实时面板" size="85%">
        <iframe 
          v-if="drawerVisible"
          :src="iframeSrc" 
          width="100%" 
          height="100%" 
          frameborder="0"
          class="metacube-iframe"
        ></iframe>
      </el-drawer>
    </template>

    <script setup>
    import { ref, computed } from 'vue'

    const drawerVisible = ref(false)
    const currentDevice = ref(null)

    // 动态生成直连 URL
    const iframeSrc = computed(() => {
      if (!currentDevice.value) return ''
      // 这里的参数名需与 Step 2 中 Agent 改造的逻辑对齐
      return `/node-panel/?hostname=${currentDevice.value.ip}&port=${currentDevice.value.apiPort}&secret=${currentDevice.value.secret}`
    })

    const openNodePanel = (device) => {
      currentDevice.value = device
      drawerVisible.value = true
    }
    </script>
    
    <style scoped>
    /* 隐藏 iframe 默认的滚动条，交由内部应用自己接管 */
    .metacube-iframe {
      border: none;
      overflow: hidden;
    }
    </style>
    ```
