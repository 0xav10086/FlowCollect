<template>
  <div class="chart-container applicants" ref="containerRef">
    <div class="chart-container-header">
      <h2 :title="isFake ? '默认是接收真实的下行数据，但是现在好像出现了一些问题，所以切换到测试数据，虽然没什么用，但至少在动:)' : 'This is demonstrating how the equipment is being used'">
        Equipment Status{{ isFake ? '( FAKE! )' : '' }}
      </h2>
      <span>Active Devices</span>
    </div>
    
    <div class="device-list" :class="{ 'scrollable': devices.length >= 4}" :style="{ '--item-count': devices.length }">
      <div class="device-item" v-for="dev in devices" :key="dev.name">
        <div class="device-icon" :style="{ backgroundColor: dev.color }">
          {{ dev.name.charAt(0).toUpperCase() }}
        </div>
        <div class="device-info">
          <!-- 第一行：设备名 + 连接时间 -->
          <div class="line-1">
            <span class="device-name" :title="dev.name">{{ dev.name }}</span>
            <span class="uptime">{{ dev.uptimeStr }}</span>
          </div>
          
          <!-- 第二行：dev.current/dev.total (activeConns) -->
          <div class="line-2">
            <span class="traffic-main" title="Download Traffic (Current/Total)">{{ dev.current }} / {{ dev.total }}</span>
            <span class="conns">({{ dev.activeConns }} active)</span>
          </div>

          <!-- 第三行：当前节点/直连 + 历史节点/直连 -->
          <div class="line-3">
            Now: {{ dev.curProxy }}/{{ dev.curDirect }} + Hist: {{ dev.histProxy }}/{{ dev.histDirect }}
          </div>

          <!-- 第四行：acquisitions-bar -->
          <div class="line-4 acquisitions-bar-mini">
            <span
                v-for="(seg, idx) in dev.segments"
                :key="idx" :style="{ width: seg.width + '%', backgroundColor: seg.color }"
                :title="seg.name + ': ' + seg.val"
            ></span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
// @ts-ignore
import { createDraggable } from 'animejs/draggable'

interface DeviceStat {
  name: string
  uptime: number
  uptimeStr: string
  current: string
  total: string
  activeConns: number
  color: string
  // 流量详情
  curProxy: string
  curDirect: string
  histProxy: string
  histDirect: string
  segments: { width: number, color: string, name: string, val: string }[]
}

const containerRef = ref<HTMLElement | null>(null)
const devices = ref<DeviceStat[]>([])
const isFake = ref(false)
let pollTimer: number | null = null

// 简单的颜色生成与持久化
const getRandomColor = () => `hsl(${Math.floor(Math.random() * 360)}, 70%, 60%)`
const deviceColors = ref<Record<string, string>>({})
// 节点颜色 (从 FlowDistribution 读取)
const nodeColors = ref<Record<string, string>>({})
// 设备流量历史 (本地累加)
const deviceHistory = ref<Record<string, { proxy: number, direct: number }>>({})

const formatBytes = (bytes: number) => {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

const formatUptime = (seconds: number) => {
  const d = Math.floor(seconds / 86400)
  const h = Math.floor((seconds % 86400) / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  if (d > 0) return `${d}d ${h}h`
  if (h > 0) return `${h}h ${m}m`
  return `${m}m`
}

const fetchData = async () => {
  try {
    const token = localStorage.getItem('token')
    let stats: any[] = []
    let fetchSuccess = false

    // 1. Try Real API
    try {
      const res = await fetch('/api/stats', {
        headers: { 'Authorization': `Bearer ${token}` }
      })
      if (res.ok) {
        const data = await res.json()
        if (data.device_stats) {
          stats = data.device_stats
          isFake.value = false
          fetchSuccess = true
        }
      }
    } catch (e) {
      // Ignore
    }

    // 2. Fallback to Fake API
    if (!fetchSuccess) {
      const res = await fetch('/api/fake/stats', {
        headers: { 'Authorization': `Bearer ${token}` }
      })
      if (!res.ok) return
      const data = await res.json()
      stats = data.device_stats || []
      isFake.value = true
    }
    
    // 优化：在数据处理前才读取颜色，给 FlowDistribution 更多时间去生成和保存颜色
    const storedNodeColors = localStorage.getItem('flow_node_colors')
    if (storedNodeColors) {
      nodeColors.value = JSON.parse(storedNodeColors)
      console.log(`[EquipmentStatus] Loaded ${Object.keys(nodeColors.value).length} node colors from storage.`)
    } else {
      console.warn('[EquipmentStatus] No node colors found in storage yet.')
    }

    devices.value = stats.map((d: any) => {
      // 1. 设备图标颜色
      if (!deviceColors.value[d.device_name]) {
        deviceColors.value[d.device_name] = getRandomColor()
        localStorage.setItem('device_colors', JSON.stringify(deviceColors.value))
      }

      // 2. 计算当前 Proxy vs Direct
      let curProxyBytes = 0
      let curDirectBytes = 0
      const segments: any[] = []
      // 修复：fake_api 更新后，node_usage 中不再包含 value 字段，需累加 up_value + down_value
      const totalNodeBytes = d.node_usage.reduce((acc: number, n: any) => acc + (n.up_value + n.down_value), 0)

      d.node_usage.forEach((node: any) => {
        // The progress bar still shows total traffic distribution for that device
        const nodeTotal = node.up_value + node.down_value
        if (node.name === 'Direct') curDirectBytes += node.down_value
        else curProxyBytes += node.down_value

        // 构建进度条段
        const color = nodeColors.value[node.name]
        if (!color) {
          console.debug(`[EquipmentStatus] Missing color for node: ${node.name}, using default gray.`)
        }

        segments.push({
          width: totalNodeBytes > 0 ? (nodeTotal / totalNodeBytes) * 100 : 0,
          color: color || '#ccc', // 使用 storedColors
          name: node.name,
          val: formatBytes(nodeTotal)
        })
      })

      // 3. 更新历史数据
      if (!deviceHistory.value[d.device_name]) {
        deviceHistory.value[d.device_name] = { proxy: 0, direct: 0 }
      }
      deviceHistory.value[d.device_name].proxy += curProxyBytes
      deviceHistory.value[d.device_name].direct += curDirectBytes

      const hist = deviceHistory.value[d.device_name]

      return {
        name: d.device_name,
        uptime: d.uptime,
        uptimeStr: formatUptime(d.uptime),
        current: d.formatted_current_down, // Focus on download
        total: d.formatted_total_down,   // Focus on download
        activeConns: d.active_connections,
        color: deviceColors.value[d.device_name],
        curProxy: formatBytes(curProxyBytes),
        curDirect: formatBytes(curDirectBytes),
        histProxy: formatBytes(hist.proxy),
        histDirect: formatBytes(hist.direct),
        segments: segments.sort((a, b) => b.width - a.width) // 长的段在前面
      }
    }).sort((a: DeviceStat, b: DeviceStat) => b.uptime - a.uptime) // 按在线时间排序

    // 保存历史
    localStorage.setItem('device_traffic_history', JSON.stringify(deviceHistory.value))

  } catch (e) {
    console.error(e)
  }
}

onMounted(() => {
  // 加载本地颜色配置
  const storedColors = localStorage.getItem('device_colors')
  if (storedColors) deviceColors.value = JSON.parse(storedColors)

  // 加载节点颜色 (从 FlowDistribution 共享)
  const storedNodeColors = localStorage.getItem('flow_node_colors')
  if (storedNodeColors) nodeColors.value = JSON.parse(storedNodeColors)

  // 加载历史数据
  const storedHistory = localStorage.getItem('device_traffic_history')
  if (storedHistory) deviceHistory.value = JSON.parse(storedHistory)

  fetchData()
  pollTimer = window.setInterval(fetchData, 5000)

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
  if (pollTimer) clearInterval(pollTimer)
})
</script>

<style scoped>
.chart-container {
  width: 100%;
  border-radius: 10px;
  background-color: var(--app-bg-dark);
  padding: 16px;
  display: flex;
  flex-direction: column;
}

.chart-container.applicants {
  height: 100%;
  overflow: hidden; /* Let device-list handle overflow */
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
  font-size: 16px;
  line-height: 20px;
  opacity: 0.8;
  font-size: 20px;
}

.chart-container-header span {
  color: var(--app-logo);
  font-size: 16px;
  line-height: 20px;
}

.device-list {
  flex: 1;
  margin-top: 8px;
  display: flex;
  flex-direction: column;
}

/* Case 1: < 10 items (Not scrollable, distribute evenly) */
.device-list:not(.scrollable) {
  overflow-y: hidden;
}

.device-list:not(.scrollable) .device-item {
  flex: 1; /* 均匀分布撑满高度 */
}

.device-list:not(.scrollable) .device-name {
  font-size: clamp(16px, calc(26px - (var(--item-count) * 0.8px)), 24px);
  line-height: 1.4;
}

.device-list:not(.scrollable) .device-detail {
  font-size: clamp(16px, calc(24px - (var(--item-count) * 0.8px)), 20px);
}

.device-list:not(.scrollable) .device-icon {
  width: clamp(32px, calc(48px - (var(--item-count) * 1px)), 48px);
  height: clamp(32px, calc(48px - (var(--item-count) * 1px)), 48px);
  font-size: clamp(16px, calc(26px - (var(--item-count) * 0.8px)), 24px);
}

/* Case 2: >= 10 items (Scrollable) */
.device-list.scrollable {
  display: block;
  overflow-y: auto;
  scrollbar-width: thin;
}

.device-list.scrollable .device-item {
  height: 10%; /* Show approx 10 items */
  min-height: 40px;
}

.device-item {
  display: flex;
  align-items: center;
  width: 100%;
}

.device-icon {
  border-radius: 50%;
  margin-right: 10px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-weight: bold;
}

.device-info {
  flex: 1;
  overflow: hidden;
}

.line-1 {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.line-1 .device-name {
  color: var(--main-color);
  font-size: 20px;
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 70%;
}

.line-1 .uptime {
  font-size: 16px;
  color: var(--app-logo);
  background: rgba(61, 126, 255, 0.1);
  padding: 1px 4px;
  border-radius: 4px;
}

.line-2 {
  display: flex;
  align-items: center;
  font-size: 16px;
  margin-top: 2px;
  color: var(--main-color);
}

.line-3 {
  font-size: 16px;
  color: var(--secondary-color);
  margin-top: 2px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.line-4.acquisitions-bar-mini {
  width: 100%;
  height: 4px;
  border-radius: 2px;
  margin-top: 4px;
  display: flex;
  overflow: hidden;
  background: rgba(255,255,255,0.05);
}

.line-4 span {
  height: 100%;
  display: block;
}

.conns {
  opacity: 0.7;
  margin-left: 4px;
  font-size: 16px;
}
</style>