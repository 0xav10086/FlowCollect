<template>
  <div class="chart-container applicants" ref="containerRef">
    <div class="chart-container-header">
      <h2>Equipment Status</h2>
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
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
// @ts-ignore
import { createDraggable } from 'animejs/draggable'

const newApplicants = [
  { id: 1, name: 'Emma Ray', position: 'Product Designer', avatar: 'https://images.unsplash.com/photo-1587628604439-3b9a0aa7a163?ixid=MXwxMjA3fDB8MHxzZWFyY2h8MjB8fHdvbWFufGVufDB8fDB8&ixlib=rb-1.2.1&auto=format&fit=crop&w=900&q=60' },
  { id: 2, name: 'Ricky James', position: 'IOS Developer', avatar: 'https://images.unsplash.com/photo-1583195764036-6dc248ac07d9?ixid=MXwxMjA3fDB8MHxwaG90by1wYWdlfHx8fGVufDB8fHw%3D&ixlib=rb-1.2.1&auto=format&fit=crop&w=2555&q=80' },
  { id: 3, name: 'Julia Wilson', position: 'UI Developer', avatar: 'https://images.unsplash.com/photo-1450297350677-623de575f31c?ixid=MXwxMjA3fDB8MHxzZWFyY2h8MzV8fHdvbWFufGVufDB8fDB8&ixlib=rb-1.2.1&auto=format&fit=crop&w=900&q=60' },
  { id: 4, name: 'Jess Watson', position: 'Design Lead', avatar: 'https://images.unsplash.com/photo-1596815064285-45ed8a9c0463?ixid=MXwxMjA3fDB8MHxwaG90by1wYWdlfHx8fGVufDB8fHw%3D&ixlib=rb-1.2.1&auto=format&fit=crop&w=1215&q=80' },
  { id: 5, name: 'John Pellegrini', position: 'Back-End Developer', avatar: 'https://images.unsplash.com/photo-1543965170-4c01a586684e?ixid=MXwxMjA3fDB8MHxwaG90by1wYWdlfHx8fGVufDB8fHw%3D&ixlib=rb-1.2.1&auto=format&fit=crop&w=2232&q=80' }
]

const containerRef = ref<HTMLElement | null>(null)

onMounted(() => {
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
</script>

<style scoped>
.chart-container {
  width: 100%;
  border-radius: 10px;
  background-color: var(--app-bg-dark);
  padding: 16px;
}

.chart-container.applicants {
  height: 100%;
  overflow-y: auto;
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
</style>