<template>
  <div class="app-container">
    <div class="app-main">
      <div class="main-header-line">
        <h1>Applications Dashboard</h1>
        <div class="action-buttons">
          <!-- Removed menu buttons -->
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
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import ChartContainer from '@/views/ChartContainer.vue'
import LineChart from '@/views//LineChart.vue'

// 图表统计数据
const chartStats = [
  { id: 1, title: 'Applications', value: '20.5 K', percentage: 30, color: 'pink' },
  { id: 2, title: 'Shortlisted', value: '5.5 K', percentage: 60, color: 'blue' },
  { id: 3, title: 'On-hold', value: '10.5 K', percentage: 90, color: 'orange' }
]

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
  max-width: 1680px;
  margin: 0 auto;
  font-family: "Poppins", sans-serif;
  background-color: #050e2d;
  overflow: hidden;
}

.app-main {
  flex: 1;
  height: 100%;
  overflow-y: auto;
  overflow-x: hidden;
  background-color: var(--app-bg-light);
  padding: 24px;
  background: radial-gradient(circle, #051340 1%, #040f32 100%);
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