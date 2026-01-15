<template>
  <div class="app-container" :class="{ 'light-mode': !isDark }">
    <div class="app-main">
      <div class="main-header-line">
        <h1>Applications Dashboard</h1>
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
      </div>

      <!-- 图表区域 -->
      <div class="chart-row three">
        <ChartContainer
            v-for="chart in chartStats"
            :key="chart.id"
            :title="chart.title"
            :value="chart.value"
            :percentage="chart.percentage"
            :color="chart.color"
        />
      </div>

      <div class="chart-row two">
        <!-- Line Chart (Left Big Column) -->
        <LineChart />

        <!-- Right Small Column -->
        <div class="chart-container-wrapper small">
          
          <!-- Acquisitions Block -->
          <div class="chart-container">
            <div class="chart-container-header">
              <h2>Acquisitions</h2>
              <span>This month</span>
            </div>
            <div class="acquisitions-bar">
              <span class="bar-progress rejected" style="width:8%;"></span>
              <span class="bar-progress on-hold" style="width:10%;"></span>
              <span class="bar-progress shortlisted" style="width:18%;"></span>
              <span class="bar-progress applications" style="width:64%;"></span>
            </div>
            <div class="progress-bar-info">
              <span class="progress-color applications"></span>
              <span class="progress-type">Applications</span>
              <span class="progress-amount">64%</span>
            </div>
            <div class="progress-bar-info">
              <span class="progress-color shortlisted"></span>
              <span class="progress-type">Shortlisted</span>
              <span class="progress-amount">18%</span>
            </div>
            <div class="progress-bar-info">
              <span class="progress-color on-hold"></span>
              <span class="progress-type">On-hold</span>
              <span class="progress-amount">10%</span>
            </div>
            <div class="progress-bar-info">
              <span class="progress-color rejected"></span>
              <span class="progress-type">Rejected</span>
              <span class="progress-amount">8%</span>
            </div>
          </div>

          <!-- Applicants Block -->
          <div class="chart-container applicants">
            <div class="chart-container-header">
              <h2>New Applicants</h2>
              <span>Today</span>
            </div>
            <div class="applicant-line" v-for="applicant in newApplicants" :key="applicant.id">
              <img :src="applicant.avatar" alt="profile">
              <div class="applicant-info">
                <span>{{ applicant.name }}</span>
                <p>Applied for <strong>{{ applicant.position }}</strong></p>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
    <button class="reset-layout-btn" @click="resetLayout" title="Reset Layout">
      <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="feather feather-refresh-cw"><polyline points="23 4 23 10 17 10"></polyline><polyline points="1 20 1 14 7 14"></polyline><path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"></path></svg>
    </button>>
  </div>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useDark, useToggle } from '@vueuse/core'
// @ts-ignore Anime.js V4 类型定义缺失
import { animate } from 'animejs'
// @ts-ignore Anime.js V4 Draggable 模块
import { createDraggable } from 'animejs/draggable'
import ChartContainer from '@/views/ChartContainer.vue'
import LineChart from '@/views/LineChart.vue'

const router = useRouter()
const isDark = useDark()
const toggleTheme = useToggle(isDark)

const handleLogout = () => {
  localStorage.removeItem('token')
  router.push('/')
}

const goToGithub = () => {
  window.open('https://github.com/0xav10086/FlowCollect', '_blank')
}

const resetLayout = () => {
  const containers = document.querySelectorAll('.chart-container')
  containers.forEach((el) => {
    animate(el as HTMLElement, {
      x: 0,
      y: 0,
      easing: 'outElastic(1, .6)'
    })
  })
}

// 格式化流量单位
const formatBytes = (bytes: number, decimals = 2) => {
  if (!+bytes) return '0 B'
  const k = 1024
  const dm = decimals < 0 ? 0 : decimals
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB', 'PB', 'EB', 'ZB', 'YB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(dm))} ${sizes[i]}`
}

// 响应式图表数据
const chartStats = ref<any[]>([])

const fetchStats = async () => {
  try {
    const token = localStorage.getItem('token')
    const res = await fetch('/api/stats', {
      headers: { 'Authorization': `Bearer ${token}` }
    })
    const data = await res.json()
    
    const stats = []
    const subStats = data.sub_stats || []

    // --- Chart 1: Proxy 1 Usage (代理1使用量) ---
    if (subStats.length > 0) {
      const sub = subStats[0]
      const pct = sub.Total > 0 ? Math.round((sub.Used / sub.Total) * 100) : 0
      stats.push({
        id: 1,
        title: 'Proxy 1 Usage',
        value: formatBytes(sub.Used),
        percentage: pct,
        color: 'pink'
      })
    } else {
      // Fallback: 如果没有订阅，显示今日代理流量
      stats.push({
        id: 1,
        title: 'Today Proxy',
        value: formatBytes(data.summary.proxy),
        percentage: 100,
        color: 'pink'
      })
    }

    // --- Chart 2: Proxy 2 Usage (代理2使用量) OR Today's Local ---
    if (subStats.length > 1) {
      const sub = subStats[1]
      const pct = sub.Total > 0 ? Math.round((sub.Used / sub.Total) * 100) : 0
      stats.push({
        id: 2,
        title: 'Proxy 2 Usage',
        value: formatBytes(sub.Used),
        percentage: pct,
        color: 'blue'
      })
    } else {
      // Fallback: 如果没有第二个订阅，显示今日本地流量 (非常有用的数据)
      stats.push({
        id: 2,
        title: 'Today Local',
        value: formatBytes(data.summary.local),
        percentage: 100,
        color: 'blue'
      })
    }

    // --- Chart 3: Remaining (预计结算/剩余流量) ---
    if (subStats.length > 0) {
      const sub = subStats[0]
      const remaining = sub.Total - sub.Used
      // 计算剩余百分比
      const pct = sub.Total > 0 ? Math.round((remaining / sub.Total) * 100) : 0
      stats.push({
        id: 3,
        title: 'Remaining',
        value: formatBytes(remaining),
        percentage: pct,
        color: 'orange'
      })
    } else {
      // Fallback: 显示今日总流量
      stats.push({
        id: 3,
        title: 'Total Today',
        value: formatBytes(data.summary.proxy + data.summary.local),
        percentage: 100,
        color: 'orange'
      })
    }

    chartStats.value = stats
  } catch (e) {
    console.error('Failed to fetch stats:', e)
    // 出错时保持空或显示默认
  }
}

onMounted(() => {
  fetchStats()
  
  const containers = document.querySelectorAll('.chart-container')

  containers.forEach((el) => {
    const element = el as HTMLElement
    element.style.cursor = 'grab'

    // 使用 Anime.js V4 的 createDraggable
    createDraggable(element, {
      container: '.app-main',
      // 拖拽开始
      onDown: () => {
      element.style.cursor = 'grabbing'
      element.style.zIndex = '1000'
      },
      // 拖拽结束 (释放)
      onUp: () => {
        element.style.cursor = 'grab'
        element.style.zIndex = ''
      }
    } as any)
  })
})

// 新申请人数据
const newApplicants = [
  { id: 1, name: 'Emma Ray', position: 'Product Designer', avatar: 'https://images.unsplash.com/photo-1587628604439-3b9a0aa7a163?ixid=MXwxMjA3fDB8MHxzZWFyY2h8MjB8fHdvbWFufGVufDB8fDB8&ixlib=rb-1.2.1&auto=format&fit=crop&w=900&q=60' },
  { id: 2, name: 'Ricky James', position: 'IOS Developer', avatar: 'https://images.unsplash.com/photo-1583195764036-6dc248ac07d9?ixid=MXwxMjA3fDB8MHxwaG90by1wYWdlfHx8fGVufDB8fHw%3D&ixlib=rb-1.2.1&auto=format&fit=crop&w=2555&q=80' },
  { id: 3, name: 'Julia Wilson', position: 'UI Developer', avatar: 'https://images.unsplash.com/photo-1450297350677-623de575f31c?ixid=MXwxMjA3fDB8MHxzZWFyY2h8MzV8fHdvbWFufGVufDB8fDB8&ixlib=rb-1.2.1&auto=format&fit=crop&w=900&q=60' },
  { id: 4, name: 'Jess Watson', position: 'Design Lead', avatar: 'https://images.unsplash.com/photo-1596815064285-45ed8a9c0463?ixid=MXwxMjA3fDB8MHxwaG90by1wYWdlfHx8fGVufDB8fHw%3D&ixlib=rb-1.2.1&auto=format&fit=crop&w=1215&q=80' },
  { id: 5, name: 'John Pellegrini', position: 'Back-End Developer', avatar: 'https://images.unsplash.com/photo-1543965170-4c01a586684e?ixid=MXwxMjA3fDB8MHxwaG90by1wYWdlfHx8fGVufDB8fHw%3D&ixlib=rb-1.2.1&auto=format&fit=crop&w=2232&q=80' }
]
</script>

<style scoped>
@import url("https://fonts.googleapis.com/css2?family=Poppins:wght@300;400;500&display=swap");

.app-container {
  /* Dark Mode Variables (Default) */
  --app-bg-dark: #01081f;
  --app-bg-light: #151c32;
  --app-logo: #3d7eff;
  --main-color: #fff;
  --secondary-color: #5e6a81;
  --list-item-hover: #0c1635;

  width: 100%;
  height: 100%;
  display: flex;
  position: relative;
  margin: 0 auto;
  font-family: "Poppins", sans-serif;
  background-color: #050e2d;
  overflow: hidden;
  transition: all 0.3s ease;
}

/* Light Mode Variables */
.app-container.light-mode {
  --app-bg-dark: #f0f2f5;
  --app-bg-light: #ffffff;
  --app-logo: #3d7eff;
  --main-color: #1f2937;
  --secondary-color: #6b7280;
  --list-item-hover: #e5e7eb;
  background-color: #f3f4f6;
}

.app-main {
  flex: 1;
  height: 100%;
  overflow-y: auto;
  overflow-x: hidden;
  background-color: var(--app-bg-light);
  padding: 24px;
  background: radial-gradient(circle, #051340 1%, #040f32 100%);
  
  /* Hide Scrollbar */
  scrollbar-width: none; /* Firefox */
  -ms-overflow-style: none;  /* IE 10+ */
}

.app-main::-webkit-scrollbar {
  display: none; /* Chrome/Safari */
}

.app-container.light-mode .app-main {
  background: var(--app-bg-light);
}

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
}

.reset-layout-btn {
  position: absolute;
  bottom: 32px;
  right: 32px;
  width: 48px;
  height: 48px;
  border-radius: 50%;
  background-color: var(--app-logo);
  color: #fff;
  border: none;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
  cursor: pointer;
  transition: transform 0.2s, background-color 0.2s;
  z-index: 2000;
}

.reset-layout-btn:hover {
  transform: scale(1.1) rotate(180deg);
  background-color: #2c6ae4;
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

.chart-row {
  display: flex;
  justify-content: space-between;
  margin: 0 -8px;
}

.chart-row.three {
  margin-bottom: 32px;
}

.chart-row.three > * {
  width: 33.3%;
}

.chart-row.two {
  /* Flex layout handled by children */
}

.chart-row.two .small {
  width: 33.3%;
  display: flex;
  flex-direction: column;
}

.chart-row.two .small .chart-container {
  width: 100%;
  display: flex;
  flex-direction: column;
}

.chart-row.two .small .chart-container + .chart-container {
  margin-top: 16px;
}

/* Chart Container Generic Styles */
.chart-container {
  width: 100%;
  border-radius: 10px;
  background-color: var(--app-bg-dark);
  padding: 16px;
}

.chart-container-wrapper {
  padding: 8px;
}

.chart-container-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
  margin-bottom: 12px;
}

.chart-container-header h2 {
  margin: 0;
  color: var(--main-color);
  font-size: 12px;
  line-height: 16px;
  opacity: 0.8;
}

.chart-container-header span {
  color: var(--app-logo);
  font-size: 12px;
  line-height: 16px;
}

/* Acquisitions Bar */
.acquisitions-bar {
  width: 100%;
  height: 4px;
  border-radius: 4px;
  margin-top: 16px;
  margin-bottom: 8px;
  display: flex;
}

.bar-progress {
  height: 4px;
  display: inline-block;
}

.bar-progress.applications { background-color: #ff7dcb; }
.bar-progress.shortlisted { background-color: #00cfde; }
.bar-progress.on-hold { background-color: #fdac42; }
.bar-progress.rejected { background-color: #ff5c5c; }

.progress-bar-info {
  display: flex;
  align-items: center;
  margin-top: 8px;
  width: 100%;
}

.progress-color {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  margin-right: 8px;
}

.progress-color.applications { background-color: #ff7dcb; }
.progress-color.shortlisted { background-color: #00cfde; }
.progress-color.on-hold { background-color: #fdac42; }
.progress-color.rejected { background-color: #ff5c5c; }

.progress-type {
  color: var(--secondary-color);
  font-size: 12px;
  line-height: 16px;
}

.progress-amount {
  color: var(--secondary-color);
  font-size: 12px;
  line-height: 16px;
  margin-left: auto;
}

/* Applicants List */
.chart-container.applicants {
  max-height: 336px;
  overflow-y: auto;
}

.applicant-line {
  display: flex;
  align-items: center;
  width: 100%;
  margin-top: 12px;
}

.applicant-line img {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  object-fit: cover;
  margin-right: 10px;
  flex-shrink: 0;
}

.applicant-info span {
  color: var(--main-color);
  font-size: 14px;
  line-height: 16px;
}

.applicant-info p {
  margin: 4px 0;
  font-size: 12px;
  line-height: 16px;
  color: var(--secondary-color);
}

.applicant-info strong {
  color: #fff;
  font-weight: 500;
}

/* Responsive */
@media screen and (max-width: 1180px) {
  .chart-row.two {
    flex-direction: column;
  }
  .chart-row.two .small {
    width: 100%;
    display: flex;
    flex-direction: row;
    justify-content: space-between;
  }
  .chart-row.two .small .chart-container {
    width: calc(50% - 8px);
  }
  .chart-row.two .small .chart-container + .chart-container {
    margin-top: 0;
  }
}

@media screen and (max-width: 650px) {
  .chart-row.three {
    flex-direction: column;
  }
  .chart-row.three > * {
    width: 100%;
  }
}

@media screen and (max-width: 520px) {
  .chart-row.two .small {
    flex-direction: column;
  }
  .chart-row.two .small .chart-container {
    width: 100%;
  }
  .chart-row.two .small .chart-container + .chart-container {
    margin-top: 16px;
  }
}
</style>