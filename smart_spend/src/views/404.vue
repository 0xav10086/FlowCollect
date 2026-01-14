<template>
  <section class="error-body">
    <video
      preload="auto"
      class="background"
      :src="videoSrc"
      autoplay
      muted
      loop
      @error="handleVideoError"
      @loadeddata="handleVideoSuccess"
    ></video>
    <div class="message">
      <h1 t="404">404</h1>
      <div class="bottom">
        <p>You have lost your way</p>
        <router-link to="/">return home</router-link>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { ref } from 'vue'
// 引入本地视频资源 (请确保 src/assets/err.mp4 文件存在，否则 Vite 会报错)
import localVideoPath from '@/assets/err.mp4'

const remoteVideoUrl = 'https://s3-us-west-2.amazonaws.com/s.cdpn.io/396624/err.mp4'
const videoSrc = ref(remoteVideoUrl)

const handleVideoError = (e: Event) => {
  console.warn('⚠️ 远程视频连接失败 (Remote Video Failed):', remoteVideoUrl)
  console.debug('错误详情 (Error Details):', e)
  
  // 如果当前不是本地视频，则切换到本地视频
  if (videoSrc.value !== localVideoPath) {
    console.log('🔄 正在切换到本地资源 (Switching to Local):', localVideoPath)
    videoSrc.value = localVideoPath
  }
}

const handleVideoSuccess = () => {
  console.log('✅ 视频资源加载成功 (Video Loaded):', videoSrc.value)
}
</script>

<style lang="scss">
@import '../assets/404.scss';

// 强制修正高度为视口高度，解决 Vue 中可能出现的高度塌陷问题
.error-body {
  height: 100vh;
  width: 100vw;
  overflow: hidden;
}
</style>