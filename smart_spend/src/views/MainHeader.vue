<template>
  <div class="main-header-line">
    <h1 @click="showNodeSelector" title="获取订阅配置链接" style="cursor: pointer;">FlowCollect Dashboard</h1>
    <div class="action-buttons">
      <button class="action-btn" @click="goToGithub" title="GitHub">
        <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="feather feather-github"><path d="M9 19c-5 1.5-5-2.5-7-3m14 6v-3.87a3.37 3.37 0 0 0-.94-2.61c3.14-.35 6.44-1.54 6.44-7A5.44 5.44 0 0 0 20 4.77 5.07 5.07 0 0 0 19.91 1S18.73.65 16 2.48a13.38 13.38 0 0 0-7 0C6.27.65 5.09 1 5.09 1A5.07 5.07 0 0 0 5 4.77a5.44 5.44 0 0 0-1.5 3.78c0 5.42 3.3 6.61 6.44 7A3.37 3.37 0 0 0 9 18.13V22"></path></svg>
      </button>
      <button class="action-btn" @click="toggleTheme()" title="Switch Theme">
        <svg v-if="isDark" xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="feather feather-sun"><circle cx="12" cy="12" r="5"></circle><line x1="12" y1="1" x2="12" y2="3"></line><line x1="12" y1="21" x2="12" y2="23"></line><line x1="4.22" y1="4.22" x2="5.64" y2="5.64"></line><line x1="18.36" y1="18.36" x2="19.78" y2="19.78"></line><line x1="1" y1="12" x2="3" y2="12"></line><line x1="21" y1="12" x2="23" y2="12"></line><line x1="4.22" y1="19.78" x2="5.64" y2="18.36"></line><line x1="18.36" y1="5.64" x2="19.78" y2="4.22"></line></svg>
        <svg v-else xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="feather feather-moon"><path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"></path></svg>
      </button>
      <button class="action-btn" @click="handleLogout" title="Log Out">
        <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="feather feather-log-out"><path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"></path><polyline points="16 17 21 12 16 7"></polyline><line x1="21" y1="12" x2="9" y2="12"></line></svg>
      </button>
    </div>

    <!-- 节点选择弹窗 -->
    <el-dialog v-model="dialogVisible" title="获取订阅配置" width="400px">
      <div class="node-list">
        <el-button class="node-btn" type="success" @click="copySubscription">
          复制 Clash 总订阅链接
        </el-button>
        <el-button class="node-btn trigger-update-btn" type="primary" @click="triggerUpdate">
          触发手动更新节点与规则
        </el-button>
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useDark, useToggle } from '@vueuse/core'
import { ElMessage, ElMessageBox } from 'element-plus'

const router = useRouter()
const isDark = useDark()
const toggleTheme = useToggle(isDark)

const dialogVisible = ref(false)

const handleLogout = () => {
  localStorage.removeItem('token')
  router.push('/')
}

const goToGithub = () => {
  window.open('https://github.com/0xav10086/FlowCollect', '_blank')
}

const showNodeSelector = () => {
  dialogVisible.value = true
}

const copySubscription = async () => {
  // 根据业务逻辑，只提供一个总订阅链接用于 Clash 导入。
  // 其他的 YAML（如 cf_node 等）是供这个总文件通过 provider 引入的。
  const baseUrl = import.meta.env.VITE_API_URL || window.location.origin
  const subUrl = `${baseUrl}/power_by_0xav10086`

  try {
    await navigator.clipboard.writeText(subUrl)
    ElMessage.success({
      message: `总订阅链接已复制到剪贴板！\n${subUrl}`,
      duration: 3000
    })
    dialogVisible.value = false
  } catch (err) {
    ElMessage.error('复制失败，请手动复制或检查浏览器权限。')
  }
}

const triggerUpdate = async () => {
  try {
    const confirm = await ElMessageBox.confirm(
      '此操作将在服务器后台执行节点抓取和规则合并逻辑，确认继续？',
      '触发更新',
      { confirmButtonText: '确定', cancelButtonText: '取消', type: 'warning' }
    )

    if (confirm) {
      const baseURL = import.meta.env.VITE_API_URL || ''
      const token = localStorage.getItem('token') || ''
      if (!token) {
        ElMessage.error('请先登录以获取权限！')
        router.push('/') // 重定向到登录页
        return
      }

      // 使用原生的 fetch 替代 axios，避免引入额外依赖
      const response = await fetch(`${baseURL}/api/trigger-update`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json'
        }
      })

      const data = await response.json()

      if (!response.ok) {
        throw new Error(data.error || '触发失败，请检查登录状态或服务器日志')
      }

      ElMessage.success(data.message || '更新任务已触发，请留意邮件通知')
      dialogVisible.value = false
    }
  } catch (error: any) {
    if (error !== 'cancel' && error.message !== 'cancel') {
      console.error('触发更新失败:', error)
      ElMessage.error(error.message || '触发失败，请检查网络或日志')
    }
  }
}
</script>

<style scoped>
.main-header-line {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.main-header-line h1 {
  color: var(--main-color);
  margin: 0;
  font-size: 24px;
  line-height: 32px;
  transition: opacity 0.2s;
}

.main-header-line h1:hover {
  opacity: 0.8;
}

.action-buttons {
  display: flex;
  align-items: center;
  gap: 16px;
}

.action-btn {
  background: transparent;
  border: none;
  color: var(--main-color);
  cursor: pointer;
  padding: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: opacity 0.2s;
}

.action-btn:hover {
  opacity: 0.8;
}

.node-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.node-btn {
  width: 100%;
  justify-content: center;
  text-align: center;
}

.trigger-update-btn {
  margin-top: 10px;
}
</style>
