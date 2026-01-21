<template>
  <div class="chart-container-wrapper big">
    <div class="chart-container" ref="containerRef">
      <div class="chart-container-header" @click="toggleTimeRange" style="cursor: pointer; user-select: none;">
        <h2 title="Click to switch time range">Top Active Jobs ({{ timeRangeLabel }})</h2>
        <span>{{ subTitle }}</span>
      </div>
      
      <div class="chart-body">
        <!-- Y Axis Labels -->
        <div class="y-axis-labels">
          <span v-for="(label, i) in yLabels" :key="i">{{ label }}</span>
        </div>

        <!-- Chart Area -->
        <div class="chart-wrapper" ref="chartWrapperRef">
          <svg 
            v-if="width > 0 && height > 0" 
            :viewBox="`0 0 ${width} ${height}`" 
            class="chart-svg"
            @mousemove="handleMouseMove"
            @mouseleave="handleMouseLeave"
          >
            <defs>
              <!-- Blue Gradient (Real) -->
              <linearGradient id="realGradient" x1="0%" y1="0%" x2="0%" y2="100%">
                <stop offset="0%" style="stop-color:rgba(0, 199, 214, 0.4);stop-opacity:1" />
                <stop offset="100%" style="stop-color:rgba(0, 199, 214, 0.05);stop-opacity:0" />
              </linearGradient>
              <!-- Red Gradient (Fake) -->
              <linearGradient id="fakeGradient" x1="0%" y1="0%" x2="0%" y2="100%">
                <stop offset="0%" style="stop-color:rgba(255, 92, 92, 0.4);stop-opacity:1" />
                <stop offset="100%" style="stop-color:rgba(255, 92, 92, 0.05);stop-opacity:0" />
              </linearGradient>
            </defs>

            <!-- Fake Data (Red) -->
            <path :d="fakeAreaPath" fill="url(#fakeGradient)" class="chart-area fake" style="opacity: 0;" />
            <path :d="fakeLinePath" fill="none" stroke="#ff5c5c" stroke-width="2" class="chart-line fake" stroke-linecap="round" stroke-linejoin="round" />
            
            <!-- Real Data (Blue) -->
            <path :d="realAreaPath" fill="url(#realGradient)" class="chart-area real" style="opacity: 0;" />
            <path :d="realLinePath" fill="none" stroke="#00c7d6" stroke-width="2" class="chart-line real" stroke-linecap="round" stroke-linejoin="round" />

            <!-- Points (Real) -->
            <circle
                v-for="(p, index) in realPoints"
                :key="'real-'+p.id"
                :cx="p.x"
                :cy="p.y"
                r="3"
                fill="#01081f"
                stroke="#00c7d6"
                stroke-width="2"
                class="chart-point real"
                style="opacity: 0;"
            />
            
            <!-- Points (Fake) -->
            <circle
                v-for="(p, index) in fakePoints"
                :key="'fake-'+p.id"
                :cx="p.x"
                :cy="p.y"
                r="3"
                fill="#01081f"
                stroke="#ff5c5c"
                stroke-width="2"
                class="chart-point fake"
                style="opacity: 0;"
            />

            <!-- Hover Indicator -->
            <line 
              v-if="hoverInfo.visible"
              :x1="hoverInfo.x" 
              y1="0" 
              :x2="hoverInfo.x" 
              :y2="height" 
              stroke="rgba(255,255,255,0.2)" 
              stroke-dasharray="4"
            />
          </svg>

          <!-- Tooltip -->
          <div v-if="hoverInfo.visible" class="chart-tooltip" :class="hoverInfo.align" :style="{ left: hoverInfo.x + 'px', top: '10px' }">
            <div class="tooltip-time">{{ hoverInfo.time }}</div>
            <div class="tooltip-item real">
              <span class="dot"></span> Real: {{ hoverInfo.realVal }}
            </div>
            <div class="tooltip-item fake">
              <span class="dot"></span> Fake: {{ hoverInfo.fakeVal }}
            </div>
          </div>

          <!-- X Axis Labels -->
          <div class="x-axis-labels" v-if="width > 0">
            <span v-for="(label, i) in xLabels" :key="i" :style="{ left: label.x + 'px' }">
              {{ label.text }}
            </span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed, watch, nextTick } from 'vue'
// @ts-ignore Anime.js V4
import { animate, stagger } from 'animejs'
// @ts-ignore
import { createDraggable } from 'animejs/draggable'

// --- Types ---
interface DataPoint {
  id: number
  val: number // Parsed numeric value (bytes)
  x: number
  y: number
}

// --- State ---
const containerRef = ref<HTMLElement | null>(null)
const chartWrapperRef = ref<HTMLElement | null>(null)
const width = ref(0)
const height = ref(0)

const realPoints = ref<DataPoint[]>([])
const fakePoints = ref<DataPoint[]>([])

const timeRangeIndex = ref(0)
const isAnimating = ref(false)
const hoverInfo = ref({ visible: false, x: 0, time: '', realVal: '', fakeVal: '', align: 'center' })
const now = ref(new Date()) // Reactive time for X-axis

let resizeObserver: ResizeObserver | null = null
let pollTimer: number | null = null
let uid = 0

// Configuration for Time Ranges
const ranges = [
  // interval: 刷新频率, duration: X轴总跨度(毫秒)
  { label: '1 Min', sub: 'Last 60 Seconds', interval: 5000, points: 12, duration: 60 * 1000 },
  { label: '1 Hour', sub: 'Last 60 Minutes', interval: 5000 * 12, points: 12, duration: 60 * 60 * 1000 },
  { label: '1 Day', sub: 'Last 24 Hours', interval: 5000 * 12 * 24, points: 12, duration: 24 * 60 * 60 * 1000 }
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

const formatBytes = (bytes: number) => {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

// Fetch Real Data (Blue)
const fetchRealStats = async (): Promise<number> => {
  try {
    const token = localStorage.getItem('token')
    const res = await fetch('/api/stats', {
      headers: { 'Authorization': `Bearer ${token}` }
    })
    if (!res.ok) return 0
    const json = await res.json()
    // 使用 summary.proxy 作为实时数据点
    return json.summary ? json.summary.proxy : 0
  } catch {
    return 0
  }
}

// Fetch Fake Data (Red)
const fetchFakeStats = async (): Promise<number> => {
  try {
    const token = localStorage.getItem('token')
    const res = await fetch('/api/fake/stats', {
      headers: { 'Authorization': `Bearer ${token}` }
    })
    if (!res.ok) {
      throw new Error(`HTTP error! status: ${res.status}`)
    }
    const json = await res.json()
    if (json.historical && json.historical.length > 0) {
      return parseBytes(json.historical[0].value)
    }
    return 0
  } catch (e) {
    return Math.random() * 1024 * 1024 * 100 // Fallback random
  }
}

// --- Layout & Scaling ---
const updateDimensions = () => {
  if (chartWrapperRef.value) {
    width.value = chartWrapperRef.value.clientWidth
    height.value = chartWrapperRef.value.clientHeight
    recalcPoints()
  }
}

const getMaxVal = () => {
  const maxReal = Math.max(...realPoints.value.map(p => p.val), 0)
  const maxFake = Math.max(...fakePoints.value.map(p => p.val), 0)
  return Math.max(maxReal, maxFake, 1) * 1.2 // 20% padding
}

const calculateY = (val: number, max: number) => {
  // Invert Y because SVG 0 is top
  const padding = 10
  const availableHeight = height.value - padding
  const ratio = val / max
  return height.value - (ratio * availableHeight)
}

const recalcPoints = () => {
  if (realPoints.value.length === 0) return
  const step = width.value / (ranges[timeRangeIndex.value].points - 1)
  const max = getMaxVal()

  const updateP = (p: DataPoint, i: number) => {
    p.x = i * step
    p.y = calculateY(p.val, max)
  }

  realPoints.value.forEach(updateP)
  fakePoints.value.forEach(updateP)
}

// --- Computed Paths ---
const getLinePath = (pts: DataPoint[]) => {
  if (pts.length === 0) return ''
  return pts.map((p, i) => `${i === 0 ? 'M' : 'L'} ${p.x} ${p.y}`).join(' ')
}

const getAreaPath = (pts: DataPoint[]) => {
  if (pts.length === 0) return ''
  const line = getLinePath(pts)
  const first = pts[0]
  const last = pts[pts.length - 1]
  return `${line} L ${last.x} ${height.value} L ${first.x} ${height.value} Z`
}

const realLinePath = computed(() => getLinePath(realPoints.value))
const realAreaPath = computed(() => getAreaPath(realPoints.value))
const fakeLinePath = computed(() => getLinePath(fakePoints.value))
const fakeAreaPath = computed(() => getAreaPath(fakePoints.value))

const yLabels = computed(() => {
  const max = getMaxVal() / 1.2 // Remove padding for label calculation
  return [
    formatBytes(max),
    formatBytes(max * 0.75),
    formatBytes(max * 0.5),
    formatBytes(max * 0.25),
    '0 B'
  ]
})

const xLabels = computed(() => {
  if (width.value === 0) return []
  const count = 6
  const labels = []
  const range = ranges[timeRangeIndex.value]
  const duration = range.duration

  for (let i = 0; i < count; i++) {
    // i=0 是最左边 (过去), i=count-1 是最右边 (现在)
    // 计算该标签代表的时间点
    const timeOffset = duration - (i * (duration / (count - 1)))
    const d = new Date(now.value.getTime() - timeOffset)
    
    // 格式化时间
    const timeStr = d.toLocaleTimeString('en-GB', { hour12: false, hour: '2-digit', minute: '2-digit', second: range.label === '1 Day' ? undefined : '2-digit' })

    labels.push({
      text: timeStr,
      x: (width.value / (count - 1)) * i - 20 // Center align adjustment
    })
  }
  return labels
})

// --- Interaction ---
const handleMouseMove = (e: MouseEvent) => {
  if (realPoints.value.length === 0) return
  const rect = (e.target as Element).closest('svg')?.getBoundingClientRect()
  if (!rect) return
  
  const mouseX = e.clientX - rect.left
  const step = width.value / (ranges[timeRangeIndex.value].points - 1)
  
  // Find closest index
  let index = Math.round(mouseX / step)
  if (index < 0) index = 0
  if (index >= realPoints.value.length) index = realPoints.value.length - 1

  const realP = realPoints.value[index]
  const fakeP = fakePoints.value[index]
  
  // Get time label
  const range = ranges[timeRangeIndex.value]
  const duration = range.duration
  const timeOffset = duration - (index * (duration / (ranges[timeRangeIndex.value].points - 1)))
  const d = new Date(new Date().getTime() - timeOffset)
  const timeStr = d.toLocaleTimeString('en-GB')

  // Calculate alignment to prevent clipping
  let align = 'center'
  if (index === 0) align = 'left'
  else if (index === realPoints.value.length - 1) align = 'right'

  hoverInfo.value = {
    visible: true,
    x: realP.x,
    time: timeStr,
    realVal: formatBytes(realP.val),
    fakeVal: formatBytes(fakeP.val),
    align
  }
}

const handleMouseLeave = () => {
  hoverInfo.value.visible = false
}

// --- Animations ---

// 1. Initial Load Animation
const playInitialAnimation = async () => {
  isAnimating.value = true

  // Reset styles
  const pointEls = document.querySelectorAll('.chart-point')
  const lineEls = document.querySelectorAll('.chart-line')
  const areaEls = document.querySelectorAll('.chart-area')

  if(lineEls.length === 0) return

  // Step 1: Show points left to right (1s)
  await animate(pointEls, {
    opacity: [0, 1], // 出现
    scale: [0.33, 1], // 直径 1 -> 3 (r=1.5, so 0.33*3 ≈ 1, 1*3 = 3)
    delay: stagger(1000 / pointEls.length), // 依次显示
    duration: 800,
    easing: 'easeOutElastic(1, .6)'
  }).finished

  // Step 2: Connect lines (1s)
  const lineAnims = Array.from(lineEls).map((el) => {
    const path = el as SVGPathElement
    const len = path.getTotalLength() || 1000
    path.style.strokeDasharray = `${len}`
    path.style.strokeDashoffset = `${len}`
    return animate(path, {
      strokeDashoffset: [len, 0],
      duration: 1000,
      easing: 'easeInOutQuad'
    }).finished
  })
  
  await Promise.all(lineAnims)
  
  // Fix: Remove stroke-dash styles so new segments are visible immediately
  lineEls.forEach((el) => {
    (el as HTMLElement).style.strokeDasharray = 'none';
    (el as HTMLElement).style.strokeDashoffset = '0';
  })

  // Wait for lines
  await new Promise(r => setTimeout(r, 1000))

  // Step 3: Darken area (区域颜色加深)
  animate(areaEls, {
    opacity: [0, 0.6], // 加深到 0.6
    duration: 800,
    easing: 'easeOutQuad'
  }).finished

  isAnimating.value = false
  startPolling()
}

// 2. Update Animation
const updateChart = async () => {
  if (isAnimating.value) return
  isAnimating.value = true

  // Update time for X-axis labels
  now.value = new Date()

  // Fetch new data
  const [realVal, fakeVal] = await Promise.all([fetchRealStats(), fetchFakeStats()])

  // Prepare new point state
  const step = width.value / (ranges[timeRangeIndex.value].points - 1)
  
  const createPoint = (val: number) => ({
    id: uid++, val, x: width.value + step, y: height.value
  })

  const newReal = createPoint(realVal)
  const newFake = createPoint(fakeVal)

  // Add to array temporarily to calculate Y scale
  const tempReal = [...realPoints.value, newReal]
  const tempFake = [...fakePoints.value, newFake]
  
  const maxReal = Math.max(...tempReal.map(p => p.val), 0)
  const maxFake = Math.max(...tempFake.map(p => p.val), 0)
  const max = Math.max(maxReal, maxFake, 1) * 1.2

  // Update Y for new point
  newReal.y = calculateY(realVal, max)
  newFake.y = calculateY(fakeVal, max)

  // Push new point to reactive array
  realPoints.value.push(newReal)
  fakePoints.value.push(newFake)

  // 等待 DOM 更新，以便获取新生成的点元素
  await nextTick()

  // 获取 DOM 元素
  const allRealPoints = document.querySelectorAll('.chart-point.real')
  const allFakePoints = document.querySelectorAll('.chart-point.fake')
  
  // 这里的逻辑是：数组长度现在是 13 (12旧 + 1新)
  // index 0 是要移除的最左侧点
  // index 12 (length-1) 是新加入的最右侧点
  const oldRealPoint = allRealPoints[0]
  const newRealPoint = allRealPoints[allRealPoints.length - 1]
  const oldFakePoint = allFakePoints[0]
  const newFakePoint = allFakePoints[allFakePoints.length - 1]

  // 动画配置
  const animProps = {
    x: (p: DataPoint, i: number) => (i * step) - step,
    y: (p: DataPoint) => calculateY(p.val, max),
    duration: 1000,
    easing: 'easeInOutQuad'
  }

  // 并行执行所有动画
  await Promise.all([
    // 1. 数据点位移 (Vue 响应式数据驱动 cx/cy)
    animate(realPoints.value, animProps).finished,
    animate(fakePoints.value, animProps).finished,

    // 2. 最左侧点：由大变小 (DOM 样式驱动)
    animate([oldRealPoint, oldFakePoint], {
      scale: [1, 0.33],
      opacity: [1, 0],
      duration: 1000,
      easing: 'easeInOutQuad'
    }).finished,

    // 3. 最右侧新点：由小变大 (DOM 样式驱动)
    animate([newRealPoint, newFakePoint], {
      scale: [0.33, 1], // 直径 1 -> 3
      opacity: [0, 1],  // 修复：新点必须显式设置透明度为 1
      duration: 1000,
      easing: 'easeOutElastic(1, .6)'
    }).finished
  ])

  // Remove the first point (now off-screen)
  realPoints.value.shift()
  fakePoints.value.shift()

  // Reset X coordinates to canonical positions (0, step, 2*step...)
  const resetX = (p: DataPoint, i: number) => { p.x = i * step }
  realPoints.value.forEach(resetX)
  fakePoints.value.forEach(resetX)

  // 动画完成，区域颜色加深一下作为反馈
  const areaEls = document.querySelectorAll('.chart-area')
  animate(areaEls, {
    opacity: [0.6, 0.8, 0.6], // 脉冲效果
    duration: 600,
    easing: 'easeInOutSine'
  })

  isAnimating.value = false
}

// --- Lifecycle ---
const initData = async () => {
  stopPolling()
  realPoints.value = []
  fakePoints.value = []
  now.value = new Date()

  // Generate/Fetch initial 12 points
  const count = ranges[timeRangeIndex.value].points
  const step = width.value / (count - 1)

  for (let i = 0; i < count; i++) {
    const [realVal, fakeVal] = await Promise.all([fetchRealStats(), fetchFakeStats()])
    const createP = (val: number) => ({ id: uid++, val, x: i * step, y: height.value })
    realPoints.value.push(createP(realVal))
    fakePoints.value.push(createP(fakeVal))
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
  if (chartWrapperRef.value) {
    resizeObserver = new ResizeObserver(() => {
      updateDimensions()
    })
    resizeObserver.observe(chartWrapperRef.value)
    updateDimensions()
    initData()

    // 初始化拖拽
    // 修复：绑定到最外层容器，而不是内部图表
    const element = containerRef.value
    if (!element) return
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
  if (resizeObserver) resizeObserver.disconnect()
})
</script>

<style scoped>
.chart-container-wrapper.big {
  position: fixed;
  top: 18%;
  left: 1%;
  width: 70%;
  bottom: 1%;
  z-index: 10;
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

.chart-body {
  flex: 1;
  display: flex;
  width: 100%;
  overflow: hidden;
}

.y-axis-labels {
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  padding-right: 10px;
  padding-bottom: 20px; /* Align with chart area above X axis */
  color: #5e6a81;
  font-size: 10px;
  text-align: right;
  min-width: 50px;
}

.chart-wrapper {
  flex: 1;
  position: relative;
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
}

.chart-svg {
  width: 100%;
  height: 100%;
  overflow: visible;
  cursor: crosshair;
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
  transform: translateX(-50%);
}

.chart-tooltip {
  position: absolute;
  background: rgba(1, 8, 31, 0.9);
  border: 1px solid #3d7eff;
  border-radius: 4px;
  padding: 8px;
  pointer-events: none;
  z-index: 10;
  transform: translateX(-50%);
  white-space: nowrap;
}

.chart-tooltip.left {
  transform: translateX(0);
}

.chart-tooltip.right {
  transform: translateX(-100%);
}

.tooltip-time {
  color: #fff;
  font-size: 12px;
  margin-bottom: 4px;
  border-bottom: 1px solid rgba(255,255,255,0.1);
  padding-bottom: 2px;
}

.tooltip-item {
  font-size: 11px;
  display: flex;
  align-items: center;
  margin-top: 2px;
}

.tooltip-item.real { color: #00c7d6; }
.tooltip-item.fake { color: #ff5c5c; }

.dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  margin-right: 6px;
  background-color: currentColor;
  transform-box: fill-box;
  transform-origin: center;
}
</style>