<template>
  <div class="chart-container-wrapper">
    <div class="chart-container">
      <div class="chart-info-wrapper">
        <h2>{{ title }}</h2>
        <span>{{ value }}</span>
      </div>
      <div class="chart-svg">
        <svg viewBox="0 0 36 36" :class="['circular-chart', color]">
          <path class="circle-bg" :d="circlePath" />
          <path
              class="circle"
              :stroke-dasharray="`${percentage}, 100`"
              :d="circlePath"
          />
          <text x="18" y="20.35" class="percentage">{{ percentage }}%</text>
        </svg>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
interface Props {
  title: string
  value: string
  percentage: number
  color: 'pink' | 'blue' | 'orange'
}

defineProps<Props>()

const circlePath = "M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831"
</script>

<style scoped>
.chart-container-wrapper {
  padding: 8px;
}

.chart-container {
  width: 100%;
  border-radius: 10px;
  background-color: var(--app-bg-dark, #01081f);
  padding: 16px;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.chart-info-wrapper {
  flex-shrink: 0;
  flex-basis: 120px;
}

.chart-info-wrapper h2 {
  color: var(--secondary-color, #5e6a81);
  font-size: 12px;
  line-height: 16px;
  font-weight: 600;
  text-transform: uppercase;
  margin: 0 0 8px 0;
}

.chart-info-wrapper span {
  color: var(--main-color, #fff);
  font-size: 24px;
  line-height: 32px;
  font-weight: 500;
}

.chart-svg {
  position: relative;
  max-width: 90px;
  min-width: 40px;
  flex: 1;
}

.circle-bg {
  fill: none;
  stroke: #eee;
  stroke-width: 1.2;
}

.circle {
  fill: none;
  stroke-width: 1.6;
  stroke-linecap: round;
  animation: progress 1s ease-out forwards;
}

.circular-chart.orange .circle {
  stroke: #ff9f00;
}
.circular-chart.orange .circle-bg {
  stroke: #776547;
}

.circular-chart.blue .circle {
  stroke: #00cfde;
}
.circular-chart.blue .circle-bg {
  stroke: #557b88;
}

.circular-chart.pink .circle {
  stroke: #ff7dcb;
}
.circular-chart.pink .circle-bg {
  stroke: #6f5684;
}

.percentage {
  fill: var(--main-color, #fff);
  font-size: 0.5em;
  text-anchor: middle;
  font-weight: 400;
}

@keyframes progress {
  0% {
    stroke-dasharray: 0 100;
  }
}
</style>