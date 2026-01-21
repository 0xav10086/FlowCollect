<template>
  <div class="app-container" :class="{ 'light-mode': !isDark }">
    <div class="app-main">
      <MainHeader />

      <!-- 图表区域 -->
      <StatsOverview />

      <!-- Line Chart (Left Big Column) -->
      <LineChart />

      <!-- Right Small Column -->
      <div class="chart-container-wrapper small">
        <!-- Flow Distribution Block -->
        <FlowDistribution style="flex: 1; min-height: 0;" />

        <!-- Equipment Status Block -->
        <EquipmentStatus style="flex: 1; min-height: 0;" />
      </div>
    </div>
    <button class="reset-layout-btn" @click="resetLayout" title="Reset Layout">
      <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="feather feather-refresh-cw"><polyline points="23 4 23 10 17 10"></polyline><polyline points="1 20 1 14 7 14"></polyline><path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"></path></svg>
    </button>>
  </div>
</template>

<script setup lang="ts">
import { useDark } from '@vueuse/core'
// @ts-ignore Anime.js V4 类型定义缺失
import { animate } from 'animejs'
import LineChart from '@/views/LineChart.vue'
import MainHeader from '@/views/MainHeader.vue'
import StatsOverview from '@/views/StatsOverview.vue'
import FlowDistribution from '@/views/FlowDistribution.vue'
import EquipmentStatus from '@/views/EquipmentStatus.vue'

const isDark = useDark()

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
  height: 100vh;
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

.chart-row {
  display: flex;
  justify-content: space-between;
  margin: 0 -8px;
}

.chart-container-wrapper.small {
  position: fixed;
  top: 18%;
  right: 1%;
  width: 27%;
  bottom: 1%;
  z-index: 10;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

/* Responsive */
@media screen and (max-width: 1180px) {
}

@media screen and (max-width: 650px) {
}
</style>