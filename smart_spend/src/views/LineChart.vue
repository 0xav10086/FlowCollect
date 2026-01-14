<template>
  <div class="chart-container-wrapper big">
    <div class="chart-container">
      <div class="chart-container-header">
        <h2>Top Active Jobs</h2>
        <span>Last 30 days</span>
      </div>
      <div class="line-chart">
        <canvas ref="chartCanvas"></canvas>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import Chart from 'chart.js/auto'

const chartCanvas = ref<HTMLCanvasElement | null>(null)
let chartInstance: Chart | null = null

onMounted(() => {
  if (!chartCanvas.value) return

  const ctx = chartCanvas.value.getContext('2d')
  if (!ctx) return

  const gradient = ctx.createLinearGradient(0, 0, 0, 450)
  gradient.addColorStop(0, 'rgba(0, 199, 214, 0.32)')
  gradient.addColorStop(0.3, 'rgba(0, 199, 214, 0.1)')
  gradient.addColorStop(1, 'rgba(0, 199, 214, 0)')

  const data = {
    labels: ['January', 'February', 'March', 'April', 'May', 'June', 'July', 'August', 'September', 'October', 'November', 'December'],
    datasets: [{
      label: 'Applications',
      backgroundColor: gradient,
      pointBackgroundColor: '#00c7d6',
      borderWidth: 1,
      borderColor: '#0e1a2f',
      data: [60, 45, 80, 30, 35, 55, 25, 80, 40, 50, 80, 50],
      fill: true,
      tension: 0.4
    }]
  }

  const options = {
    responsive: true,
    maintainAspectRatio: false,
    animation: {
      easing: 'easeInOutQuad',
      duration: 520
    },
    scales: {
      y: {
        ticks: {
          color: '#5e6a81'
        },
        grid: {
          color: 'rgba(200, 200, 200, 0.08)',
          lineWidth: 1
        }
      },
      x: {
        ticks: {
          color: '#5e6a81'
        },
        grid: {
          display: false
        }
      }
    },
    plugins: {
      legend: {
        display: false
      },
      tooltip: {
        titleFont: {
          family: 'Poppins'
        },
        backgroundColor: 'rgba(0,0,0,0.4)',
        titleColor: 'white',
        caretSize: 5,
        cornerRadius: 2,
        padding: 10
      }
    }
  }

  chartInstance = new Chart(ctx, {
    type: 'line',
    data: data,
    options: options
  })
})

onUnmounted(() => {
  if (chartInstance) {
    chartInstance.destroy()
  }
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
}

.line-chart canvas {
  width: 100% !important;
  height: 100% !important;
}
</style>