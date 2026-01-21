<template>
  <div class="chart-container" ref="containerRef">
    <div class="chart-container-header">
      <h2>Flow Distribution</h2>
      <div class="header-info">
        <span class="total-badge" v-if="historyTraffic">Now/Total: {{ currentTraffic }} / {{ historyTraffic }}</span>
        <span class="subtitle">Fake Nodes</span>
      </div>
    </div>
    
    <!-- 堆叠进度条 -->
    <div class="acquisitions-bar">
      <span 
        v-for="(item, index) in distribution" 
        :key="index"
        class="bar-progress"
        :style="{ width: item.percentage + '%', backgroundColor: item.color }"
        :title="`${item.name}: ${item.displayString}\nFormat: Current / History (Cur% / Hist%)`"
      ></span>
    </div>

    <!-- 图例列表 -->
    <div class="legend-list" :class="{ 'scrollable': distribution.length >= 10 }" :style="{ '--item-count': distribution.length }">
      <div class="progress-bar-info" v-for="(item, index) in distribution" :key="index">
        <span class="progress-color" :style="{ backgroundColor: item.color }"></span>
        <span class="progress-type" :title="item.name">{{ item.name }}</span>
        <span 
          class="progress-amount" 
          title="Format: Current / History (Cur% / Hist%)"
        >{{ item.displayString }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
// @ts-ignore
import { createDraggable } from 'animejs/draggable'

interface NodeDist {
  name: string
  value: number
  formattedValue: string
  percentage: number
  historyPercentage: number
  color: string
  displayString: string
}

const containerRef = ref<HTMLElement | null>(null)
const distribution = ref<NodeDist[]>([])
const currentTraffic = ref('')
const historyTraffic = ref('')
// 本地持久化状态
const nodeColors = ref<Record<string, string>>({})
const nodeHistory = ref<Record<string, number>>({})

let pollTimer: number | null = null

const formatBytes = (bytes: number) => {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

// 生成随机高亮颜色 (HSL)
const getRandomColor = () => {
  const h = Math.floor(Math.random() * 360)
  return `hsl(${h}, 70%, 60%)` // 饱和度70%，亮度60%，确保在暗色背景下清晰可见
}

// 获取或生成节点颜色
const getNodeColor = (name: string) => {
  if (!nodeColors.value[name]) {
    nodeColors.value[name] = getRandomColor()
    localStorage.setItem('flow_node_colors', JSON.stringify(nodeColors.value))
  }
  return nodeColors.value[name]
}

// 加载本地存储
const loadLocalStorage = () => {
  const storedColors = localStorage.getItem('flow_node_colors')
  if (storedColors) nodeColors.value = JSON.parse(storedColors)

  const storedHistory = localStorage.getItem('flow_node_history')
  if (storedHistory) nodeHistory.value = JSON.parse(storedHistory)
}

const fetchData = async () => {
  try {
    const token = localStorage.getItem('token')
    const res = await fetch('/api/fake/stats', {
      headers: { 'Authorization': `Bearer ${token}` }
    })
    if (!res.ok) return

    const data = await res.json()
    const rawDist = data.node_distribution || []
    
    // 调试信息输出
    const nowStr = new Date().toLocaleTimeString()
    console.group(`[${nowStr}] FlowDistribution Update`)
    console.log('Raw Data:', rawDist)

    // 1. 更新历史数据 & 计算总流量
    let currentTotal = 0
    let historyTotal = 0

    rawDist.forEach((item: any) => {
      currentTotal += item.value
      // 累加历史数据
      if (!nodeHistory.value[item.name]) nodeHistory.value[item.name] = 0
      nodeHistory.value[item.name] += item.value
    })

    // 计算所有节点的历史总和
    historyTotal = Object.values(nodeHistory.value).reduce((a, b) => a + b, 0)

    // 保存历史数据到本地
    localStorage.setItem('flow_node_history', JSON.stringify(nodeHistory.value))

    currentTraffic.value = formatBytes(currentTotal)
    historyTraffic.value = formatBytes(historyTotal)
    
    // 2. 映射数据并计算百分比
    distribution.value = rawDist.map((item: any, index: number) => {
      const currentPct = currentTotal > 0 ? Math.round((item.value / currentTotal) * 100) : 0
      const histVal = nodeHistory.value[item.name] || 0
      const histPct = historyTotal > 0 ? Math.round((histVal / historyTotal) * 100) : 0
      
      return {
        name: item.name,
        value: item.value,
        formattedValue: formatBytes(item.value),
        percentage: currentPct,
        historyPercentage: histPct,
        color: getNodeColor(item.name),
        // 格式: data/data(percent/percent)
        displayString: `${formatBytes(item.value)}/${formatBytes(histVal)} (${currentPct}%/${histPct}%)`
      }
    }).sort((a: NodeDist, b: NodeDist) => b.value - a.value) // 按流量大小排序

    console.log('Processed Distribution:', distribution.value)
    console.groupEnd()
  } catch (e) {
    console.error('Failed to fetch distribution:', e)
  }
}

const startPolling = () => {
  // Poll every 3 seconds to match chart updates roughly
  pollTimer = window.setInterval(fetchData, 5000)
}

const stopPolling = () => {
  if (pollTimer) clearInterval(pollTimer)
}

onMounted(() => {
  loadLocalStorage()
  fetchData()
  startPolling()
  
  if (containerRef.value) {
    const element = containerRef.value
    element.style.cursor = 'grab'
    createDraggable(element, {
      container: '.app-main',
      onDown: () => { element.style.cursor = 'grabbing'; element.style.zIndex = '1000' },
      onUp: () => { element.style.cursor = 'grab'; element.style.zIndex = '' }
    } as any)
  }
})

onUnmounted(() => {
  stopPolling()
})
</script>

<style scoped>
.chart-container {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  border-radius: 10px;
  background-color: var(--app-bg-dark);
  padding: 16px;
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

.header-info {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
}

.subtitle {
  color: var(--app-logo);
  font-size: 12px;
  line-height: 16px;
}

.total-badge {
  color: var(--main-color);
  font-size: 10px;
  font-weight: 600;
  opacity: 0.7;
}

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
  transition: width 0.5s ease;
}

.legend-list {
  flex: 1;
  margin-top: 8px;
  display: flex;
  flex-direction: column;
}

/* Case 1: < 10 items (Not scrollable, distribute evenly) */
.legend-list:not(.scrollable) {
  overflow-y: hidden;
}

.legend-list:not(.scrollable) .progress-bar-info {
  flex: 1; /* 均匀分布撑满高度 */
}

.legend-list:not(.scrollable) .progress-type,
.legend-list:not(.scrollable) .progress-amount {
  /* 自适应字号：节点越少字越大。计算公式：基准24px - (数量 * 0.8px)，限制在 14px 到 16px 之间 */
  font-size: clamp(14px, calc(24px - (var(--item-count) * 0.8px)), 22px);
  line-height: 1.4;
}

.legend-list:not(.scrollable) .progress-color {
  width: clamp(10px, calc(18px - (var(--item-count) * 0.6px)), 16px);
  height: clamp(10px, calc(18px - (var(--item-count) * 0.6px)), 16px);
}

/* Case 2: >= 10 items (Scrollable, 10 items per view) */
.legend-list.scrollable {
  display: block; /* 切换为 block 以支持滚动溢出计算 */
  overflow-y: auto;
  scrollbar-width: thin; /* Firefox */
}

.legend-list.scrollable .progress-bar-info {
  height: 10%; /* 100% / 10 = 10%，刚好显示10个 */
  min-height: 20px; /* 防止过小 */
}

.progress-bar-info {
  display: flex;
  align-items: center;
  /* margin-bottom: 12px; 移除固定间距，改由布局控制 */
  width: 100%;
}

.progress-color {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  margin-right: 8px;
  flex-shrink: 0;
}

.progress-type {
  color: var(--secondary-color);
  font-size: 12px;
  line-height: 16px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  margin-right: 8px;
}

.progress-amount {
  color: var(--secondary-color);
  font-size: 12px;
  line-height: 16px;
  margin-left: auto;
}
</style>