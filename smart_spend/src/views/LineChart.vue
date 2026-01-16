<template>
  <div class="chart-container-wrapper big">
    <div class="chart-container">
      <div class="chart-container-header" @click="toggleTimeRange" style="cursor: pointer; user-select: none;">
        <h2 title="Click to switch time range">Top Active Jobs ({{ timeRangeLabel }})</h2>
        <span>{{ subTitle }}</span>
      </div>
      <div class="line-chart" ref="chartContainer">
        <svg v-if="width > 0 && height > 0" :viewBox="`0 0 ${width} ${height}`" class="chart-svg">
          <defs>
            <linearGradient id="areaGradient" x1="0%" y1="0%" x2="0%" y2="100%">
              <stop offset="0%" style="stop-color:rgba(0, 199, 214, 0.4);stop-opacity:1" />
              <stop offset="100%" style="stop-color:rgba(0, 199, 214, 0.05);stop-opacity:0" />
            </linearGradient>
            <clipPath id="chartClip">
              <rect :width="width" :height="height" x="0" y="0" />
            </clipPath>
          </defs>

          <!-- Area (Filled) -->
          <path
              :d="areaPath"
              fill="url(#areaGradient)"
              class="chart-area"
              style="opacity: 0;"
          />

          <!-- Line -->
          <path
              :d="linePath"
              fill="none"
              stroke="#00c7d6"
              stroke-width="2"
              class="chart-line"
              stroke-linecap="round"
              stroke-linejoin="round"
          />

          <!-- Data Points -->
          <circle
              v-for="(p, index) in points"
              :key="p.id"
              :cx="p.x"
              :cy="p.y"
              r="4"
              fill="#01081f"
              stroke="#00c7d6"
              stroke-width="2"
              :data-index="index"
              class="chart-point"
              style="opacity: 0;"
          />
        </svg>

        <!-- X Axis Labels -->
        <div class="x-axis-labels" v-if="width > 0">
          <span v-for="(label, i) in xLabels" :key="i" :style="{ left: label.x + 'px' }">
            {{ label.text }}
          </span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed, watch, nextTick } from 'vue'
// @ts-ignore Anime.js V4
import { animate, stagger } from 'animejs'

// --- Types ---
interface DataPoint {
  id: number
  val: number // Parsed numeric value (bytes)
  x: number
  y: number
  raw: string // Original string (e.g. "10 MB")
}

// --- State ---
const chartContainer = ref<HTMLElement | null>(null)
const width = ref(0)
const height = ref(0)
const points = ref<DataPoint[]>([])
const timeRangeIndex = ref(0)
const isAnimating = ref(false)
let resizeObserver: ResizeObserver | null = null
let pollTimer: number | null = null
let uid = 0

// Configuration for Time Ranges
const ranges = [
  // interval: 刷新频率, duration: X轴总跨度(毫秒)
  { label: '1 Min', sub: 'Last 60 Seconds', interval: 2000, points: 12, duration: 60 * 1000 },
  { label: '1 Hour', sub: 'Last 60 Minutes', interval: 5000, points: 12, duration: 60 * 60 * 1000 },
  { label: '1 Day', sub: 'Last 24 Hours', interval: 10000, points: 12, duration: 24 * 60 * 60 * 1000 }
]

const timeRangeLabel = computed(() => ranges[timeRangeIndex.value].label)
const subTitle = computed(() => ranges[timeRangeIndex.value].sub)

// --- Helpers ---
const parseBytes = (str: string): number => {
  if (!str) return 0
  const match = str.match(/([\d\.]+)\s*([a-zA-Z]+)/)
  if (!match) return 0
  const val = parseFloat(match[1])
  const unit = match[2].toUpperCase()
  const units = ['B', 'KB', 'MB', 'GB', 'TB', 'PB']
  const exp = units.indexOf(unit)
  return exp === -1 ? val : val * Math.pow(1024, exp)
}

const fetchData = async (): Promise<number> => {
  try {
    const token = localStorage.getItem('token')
    const res = await fetch('/api/fake/stats', {
      headers: { 'Authorization': `Bearer ${token}` }
    })
    if (!res.ok) {
      throw new Error(`HTTP error! status: ${res.status}`)
    }
    const json = await res.json()
    // Assuming the API returns a 'historical' array, we take the latest value
    // Or if it returns 'data' with current stats.
    // Based on fake_api.go snippet, 'historical' has values.
    if (json.historical && json.historical.length > 0) {
      return parseBytes(json.historical[0].value)
    }
    return 0
  } catch (e) {
    // console.warn('Fetch failed, using random data:', e)
    return Math.random() * 1024 * 1024 * 100 // Fallback random
  }
}

// --- Layout & Scaling ---
const updateDimensions = () => {
  if (chartContainer.value) {
    width.value = chartContainer.value.clientWidth
    height.value = chartContainer.value.clientHeight
    recalcPoints()
  }
}

const getMaxVal = () => Math.max(...points.value.map(p => p.val), 1) * 1.2 // 20% padding

const calculateY = (val: number, max: number) => {
  // Invert Y because SVG 0 is top
  const padding = 20
  const availableHeight = height.value - padding * 2
  const ratio = val / max
  return height.value - padding - (ratio * availableHeight)
}

const recalcPoints = () => {
  if (points.value.length === 0) return
  const step = width.value / (ranges[timeRangeIndex.value].points - 1)
  const max = getMaxVal()

  points.value.forEach((p, i) => {
    p.x = i * step
    p.y = calculateY(p.val, max)
  })
}

// --- Computed Paths ---
const linePath = computed(() => {
  if (points.value.length === 0) return ''
  return points.value.map((p, i) => `${i === 0 ? 'M' : 'L'} ${p.x} ${p.y}`).join(' ')
})

const areaPath = computed(() => {
  if (points.value.length === 0) return ''
  const first = points.value[0]
  const last = points.value[points.value.length - 1]
  return `${linePath.value} L ${last.x} ${height.value} L ${first.x} ${height.value} Z`
})

const xLabels = computed(() => {
  if (width.value === 0) return []
  const count = 6
  const labels = []
  const now = new Date()
  const range = ranges[timeRangeIndex.value]
  const duration = range.duration

  for (let i = 0; i < count; i++) {
    // i=0 是最左边 (过去), i=count-1 是最右边 (现在)
    // 计算该标签代表的时间点
    const timeOffset = duration - (i * (duration / (count - 1)))
    const d = new Date(now.getTime() - timeOffset)
    
    // 格式化时间
    const timeStr = d.toLocaleTimeString('en-GB', { hour12: false, hour: '2-digit', minute: '2-digit', second: range.label === '1 Day' ? undefined : '2-digit' })

    labels.push({
      text: timeStr,
      x: (width.value / (count - 1)) * i - 15 // Center align adjustment
    })
  }
  return labels
})

// --- Animations ---

// 1. Initial Load Animation
const playInitialAnimation = async () => {
  isAnimating.value = true

  // Reset styles
  const pointEls = document.querySelectorAll('.chart-point')
  const lineEl = document.querySelector('.chart-line') as SVGPathElement
  const areaEl = document.querySelector('.chart-area') as SVGPathElement

  if(!lineEl) return

  // Step 1: Show points left to right (1s)
  await animate(pointEls, {
    opacity: [0, 1],
    scale: [0, 1],
    delay: stagger(1000 / points.value.length), // 依次显示，总耗时约1秒
    duration: 500,
    easing: 'outBack'
  }).finished

  // Step 2: Connect lines (1s)
  // Using stroke-dashoffset trick
  const len = lineEl.getTotalLength() || 1000
  lineEl.style.strokeDasharray = `${len}`
  lineEl.style.strokeDashoffset = `${len}`

  await animate(lineEl, {
    strokeDashoffset: [len, 0],
    duration: 1000,
    easing: 'easeInOutQuad'
  }).finished

  // Step 3: Darken area
  animate(areaEl, {
    opacity: [0, 1],
    duration: 500,
    easing: 'linear'
  }).finished

  isAnimating.value = false
  startPolling()
}

// 2. Update Animation
const updateChart = async () => {
  if (isAnimating.value) return
  isAnimating.value = true

  // Fetch new data
  const newVal = await fetchData()

  // Prepare new point state
  const step = width.value / (ranges[timeRangeIndex.value].points - 1)
  const newPointObj = {
    id: uid++,
    val: newVal,
    x: width.value + step, // Start off-screen right
    y: height.value, // Temporary Y
    raw: ''
  }

  // Add to array temporarily to calculate Y scale
  const tempPoints = [...points.value, newPointObj]
  const max = Math.max(...tempPoints.map(p => p.val), 1) * 1.2

  // Update Y for new point
  newPointObj.y = calculateY(newVal, max)

  // Push new point to reactive array
  points.value.push(newPointObj)

  // Animate:
  // 1. Shift all points X by -step
  // 2. Animate all points Y to new scale (if max changed)
  // 3. New point enters

  await animate(points.value, {
    x: (p: DataPoint, i: number) => {
      // 所有的点向左移动一个 step
      // i=0 (最左边的点) 将移动到 -step (移出屏幕)
      // i=12 (新点) 将移动到 width (最右边)
      return (i * step) - step
    },
    y: (p: DataPoint) => calculateY(p.val, max),
    duration: 1000,
    easing: 'easeInOutQuad'
  }).finished

  // Remove the first point (now off-screen)
  points.value.shift()

  // Reset X coordinates to canonical positions (0, step, 2*step...)
  // to prevent floating point drift and prepare for next cycle
  points.value.forEach((p, i) => {
    p.x = i * step
  })

  isAnimating.value = false
}

// --- Lifecycle ---
const initData = async () => {
  stopPolling()
  points.value = []

  // Generate/Fetch initial 12 points
  const count = ranges[timeRangeIndex.value].points
  const step = width.value / (count - 1)

  for (let i = 0; i < count; i++) {
    const val = await fetchData() // In reality, might want to fetch history array
    points.value.push({
      id: uid++,
      val: val,
      x: i * step,
      y: height.value, // Will be recalculated
      raw: ''
    })
  }

  recalcPoints()

  // Wait for DOM render then play animation
  nextTick(() => {
    playInitialAnimation()
  })
}

const startPolling = () => {
  stopPolling()
  pollTimer = window.setInterval(updateChart, ranges[timeRangeIndex.value].interval)
}

const stopPolling = () => {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

const toggleTimeRange = () => {
  timeRangeIndex.value = (timeRangeIndex.value + 1) % ranges.length
  initData()
}

onMounted(() => {
  if (chartContainer.value) {
    resizeObserver = new ResizeObserver(() => {
      updateDimensions()
    })
    resizeObserver.observe(chartContainer.value)
    updateDimensions()
    initData()
  }
})

onUnmounted(() => {
  stopPolling()
  if (resizeObserver) resizeObserver.disconnect()
})
</script>

<style scoped>
.chart-container-wrapper.big {
  flex: 1;
  width: 100%;
  padding: 8px;
}

.chart-container {
  width: 100%;
  height: 100%;
  border-radius: 10px;
  background-color: var(--app-bg-dark, #01081f);
  padding: 16px;
  display: flex;
  flex-direction: column;
}

.chart-container-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
  transition: opacity 0.2s;
}
.chart-container-header:hover {
  opacity: 0.8;
}

.chart-container-header h2 {
  margin: 0;
  color: var(--main-color, #fff);
  font-size: 12px;
  line-height: 16px;
  opacity: 0.8;
}

.chart-container-header span {
  color: var(--app-logo, #3d7eff);
  font-size: 12px;
  line-height: 16px;
}

.line-chart {
  flex: 1;
  position: relative;
  margin-top: 24px;
  width: 100%;
  overflow: hidden; /* Hide points moving out */
}

.chart-svg {
  width: 100%;
  height: 100%;
  overflow: visible;
}

.x-axis-labels {
  position: absolute;
  bottom: 0;
  left: 0;
  width: 100%;
  height: 20px;
  pointer-events: none;
}

.x-axis-labels span {
  position: absolute;
  bottom: 0;
  font-size: 10px;
  color: #5e6a81;
  white-space: nowrap;
}
</style>