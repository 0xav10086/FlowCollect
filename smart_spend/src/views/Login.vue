<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'

const router = useRouter()
const username = ref('')
const password = ref('')
const errorMsg = ref('')

const handleLogin = async () => {
  // 清空之前的错误信息
  errorMsg.value = ''

  try {
    // 发起请求给 Go 后端
    const res = await fetch('/api/auth', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        username: username.value,
        password: password.value
      })
    })

    const data = await res.json()

    if (res.ok) {
      // 登录成功：保存 Token 到 localStorage
      localStorage.setItem('token', data.token)
      // 跳转到首页
      router.push('/home')
    } else {
      // 登录失败：显示后端返回的错误信息
      errorMsg.value = data.error || '登录失败'
    }
  } catch (e) {
    errorMsg.value = '网络连接失败，请检查后端服务'
  }
}

const handleForgetPassword = (e: Event) => {
  e.preventDefault()
  // 跳转到不存在的路径，触发404
  router.push('/forgot-password-not-found')
}

const handleSignup = (e: Event) => {
  e.preventDefault()
  // 跳转到不存在的路径，触发404
  router.push('/signup-not-found')
}
</script>

<template>
  <!--ring div starts here-->
  <div class="login-container">
  <div class="ring">
    <i style="--clr:#00ff0a;"></i>
    <i style="--clr:#ff0057;"></i>
    <i style="--clr:#fffd44;"></i>
    <div class="login">
      <h2>Login</h2>
      <div class="inputBx">
        <!-- 绑定 v-model -->
        <input type="text" placeholder="Username" v-model="username">
      </div>
      <div class="inputBx">
        <input type="password" placeholder="Password" v-model="password">
      </div>
      <!-- 显示错误信息 -->
      <div v-if="errorMsg" class="error-text">{{ errorMsg }}</div>
      <div class="inputBx">
        <!-- 绑定点击事件 -->
        <input type="submit" value="Sign in" @click.prevent="handleLogin">
      </div>
      <div class="links">
        <a href="#" @click.prevent="handleForgetPassword">Forget Password</a>
        <a href="#" @click.prevent="handleSignup">Signup</a>
      </div>
    </div>
  </div>
  </div>
  <!--ring div ends here-->
</template>

<style scoped>
@import url("https://fonts.googleapis.com/css2?family=Quicksand:wght@300&display=swap");
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
  font-family: "Quicksand", sans-serif;
}
.login-container {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 100vh;
  background: #000;
  width: 100%;
  overflow: hidden;
}
.ring {
  position: relative;
  width: 500px;
  height: 500px;
  display: flex;
  justify-content: center;
  align-items: center;
  color: black;
}
.ring i {
  position: absolute;
  inset: 0;
  border: 2px solid #fff;
  transition: 0.5s;
}
.ring i:nth-child(1) {
  border-radius: 38% 62% 63% 37% / 41% 44% 56% 59%;
  animation: animate 6s linear infinite;
}
.ring i:nth-child(2) {
  border-radius: 41% 44% 56% 59%/38% 62% 63% 37%;
  animation: animate 4s linear infinite;
}
.ring i:nth-child(3) {
  border-radius: 41% 44% 56% 59%/38% 62% 63% 37%;
  animation: animate2 10s linear infinite;
}
.ring:hover i {
  border: 6px solid var(--clr);
  filter: drop-shadow(0 0 20px var(--clr));
}
@keyframes animate {
  0% {
    transform: rotate(0deg);
  }
  100% {
    transform: rotate(360deg);
  }
}
@keyframes animate2 {
  0% {
    transform: rotate(360deg);
  }
  100% {
    transform: rotate(0deg);
  }
}
.login {
  position: absolute;
  width: 300px;
  height: 100%;
  display: flex;
  justify-content: center;
  align-items: center;
  flex-direction: column;
  gap: 20px;
}
.login h2 {
  font-size: 2em;
  color: #fff;
}
.login .inputBx {
  position: relative;
  width: 100%;
}
.login .inputBx input {
  position: relative;
  width: 100%;
  padding: 12px 20px;
  background: transparent;
  border: 2px solid #fff;
  border-radius: 40px;
  font-size: 1.2em;
  color: #fff;
  box-shadow: none;
  outline: none;
}
.login .inputBx input[type="submit"] {
  width: 100%;
  background: #0078ff;
  background: linear-gradient(45deg, #ff357a, #fff172);
  border: none;
  cursor: pointer;
}
.login .inputBx input::placeholder {
  color: rgba(255, 255, 255, 0.75);
}
.login .links {
  position: relative;
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 20px;
}
.login .links a {
  color: #fff;
  text-decoration: none;
}
.error-text {
  color: #ff0057;
  font-size: 0.9rem;
  text-align: center;
}
</style>
