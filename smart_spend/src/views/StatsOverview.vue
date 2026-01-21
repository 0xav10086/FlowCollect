<template>
  <div class="chart-row three" ref="containerRef">
    <ChartContainer
        v-for="chart in chartStats"
        :key="chart.id"
        :title="chart.title"
        :value="chart.value"
        :percentage="chart.percentage"
        :color="chart.color"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, nextTick } from 'vue'
import ChartContainer from '@/views/ChartContainer.vue'
// @ts-ignore Anime.js V4 Draggable 模块
import { createDraggable } from 'animejs/draggable'

const containerRef = ref<HTMLElement | null>(null)
const chartStats = ref<any[]>([])

// 格式化流量单位
const formatBytes = (bytes: number, decimals = 2) => {
  if (!+bytes) return '0 B'
  const k = 1024
  const dm = decimals < 0 ? 0 : decimals
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB', 'PB', 'EB', 'ZB', 'YB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(dm))} ${sizes[i]}`
}

// 提取机场名称 (域名)
const getAirportName = (url: string) => {
  try {
    return new URL(url).hostname.replace('www.', '')
  } catch {
    return 'Proxy Provider'
  }
}

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
        title: getAirportName(sub.SubUrl),
        value: formatBytes(sub.Used),
        percentage: pct,
        color: 'pink'
      })
    } else {
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
        title: getAirportName(sub.SubUrl),
        value: formatBytes(sub.Used),
        percentage: pct,
        color: 'blue'
      })
    } else {
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
      const pct = sub.Total > 0 ? Math.round((remaining / sub.Total) * 100) : 0
      
      let title = 'Remaining Data'
      if (sub.Expire) {
        const now = Math.floor(Date.now() / 1000)
        const daysLeft = Math.ceil((sub.Expire - now) / 86400)
        title = `Expires in ${daysLeft} Days`
      }

      stats.push({
        id: 3,
        title: title,
        value: formatBytes(remaining),
        percentage: pct,
        color: 'orange'
      })
    } else {
      stats.push({
        id: 3,
        title: 'Total Today',
        value: formatBytes(data.summary.proxy + data.summary.local),
        percentage: 100,
        color: 'orange'
      })
    }

    chartStats.value = stats
    
    // 数据加载完成后初始化拖拽
    nextTick(() => {
      if (containerRef.value) {
        const containers = containerRef.value.querySelectorAll('.chart-container')
        containers.forEach((el) => {
          const element = el as HTMLElement
          element.style.cursor = 'grab'
          createDraggable(element, {
            container: '.app-main',
            onDown: () => { element.style.cursor = 'grabbing'; element.style.zIndex = '1000' },
            onUp: () => { element.style.cursor = 'grab'; element.style.zIndex = '' }
          } as any)
        })
      }
    })
  } catch (e) {
    console.error('Failed to fetch stats:', e)
  }
}

onMounted(() => {
  fetchStats()
})
</script>

<style scoped>
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

@media screen and (max-width: 650px) {
  .chart-row.three {
    flex-direction: column;
  }
  .chart-row.three > * {
    width: 100%;
  }
}
</style>