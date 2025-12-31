<template>
  <div class="login-container">
    <div class="theme-switch">
      <el-switch
          v-model="isDark"
          inline-prompt
          :active-icon="Moon"
          :inactive-icon="Sunny"
          @change="toggleDark"
          style="--el-switch-on-color: #2c2c2c; --el-switch-off-color: #f2f2f2"
      />
    </div>

    <div class="login-box">
      <div class="header">
        <el-avatar :size="60" src="src/assets/images/avatar.png" class="logo-avatar">
          RAG
        </el-avatar>
        <h2>{{ isRegister ? '创建账户' : '欢迎回来' }}</h2>
        <p class="subtitle">RAG 智能知识库系统</p>
      </div>

      <el-form :model="form" class="login-form">
        <el-form-item>
          <el-input
              v-model="form.username"
              placeholder="请输入用户名"
              :prefix-icon="User"
              size="large"
          />
        </el-form-item>

        <el-form-item>
          <el-input
              v-model="form.password"
              type="password"
              placeholder="请输入密码"
              :prefix-icon="Lock"
              show-password
              size="large"
          />
        </el-form-item>

        <el-button
            type="primary"
            class="submit-btn"
            size="large"
            :loading="loading"
            @click="handleAuth"
        >
          {{ isRegister ? '立即注册' : '登 录' }}
        </el-button>

        <div class="toggle-mode">
          <span v-if="!isRegister">
            还没有账号？ <a @click="isRegister = true">去注册</a>
          </span>
          <span v-else>
            已有账号？ <a @click="isRegister = false">去登录</a>
          </span>
        </div>
      </el-form>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { User, Lock, Moon, Sunny } from '@element-plus/icons-vue' // 引入图标
import { useDark, useToggle } from '@vueuse/core'
import { login, register } from '@/api/auth'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'

// --- 黑夜模式逻辑 ---
const isDark = useDark()
const toggleDark = useToggle(isDark)

// --- 登录逻辑 ---
const router = useRouter()
const isRegister = ref(false)
const loading = ref(false)
const form = ref({
  username: '',
  password: ''
})

const handleAuth = async () => {
  if (!form.value.username || !form.value.password) {
    ElMessage.warning('请输入用户名和密码')
    return
  }

  loading.value = true
  try {
    if (isRegister.value) {
      // 注册
      await register(form.value)
      ElMessage.success('注册成功，请登录')
      isRegister.value = false // 切换回登录页
    } else {
      // 登录
      const res = await login(form.value)
      // 后端返回 { token: "...", role: "..." }
      localStorage.setItem('token', res.token)
      localStorage.setItem('role', res.role)
      localStorage.setItem('username', form.value.username) // 简单存一下用户名

      ElMessage.success('登录成功')
      router.push('/') // 跳转到主页 (Chat页面)
    }
  } catch (error) {
    // 【核心修复】这里必须弹窗！
    console.error(error)
    // 获取后端返回的错误信息 (例如 "密码错误" 或 "用户不存在")
    const errorMsg = error.response?.data || '操作失败，请检查网络'
    ElMessage.error(errorMsg)
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  // 强制移除 html 标签上的 dark 类，确保登录页永远是亮色
  document.documentElement.classList.remove('dark')
})
</script>

<style scoped>
/* 容器：全屏背景 */
.login-container {
  height: 100vh;
  width: 100vw;
  display: flex;
  justify-content: center;
  align-items: center;
  /* 引用你的背景图 */
  background: url('@/assets/images/login-bg.jpg') no-repeat center center;
  background-size: cover;
  position: relative;
}

/* 遮罩层：让背景稍微暗一点，突显文字 */
.login-container::before {
  content: "";
  position: absolute;
  top: 0; left: 0; width: 100%; height: 100%;
  background: rgba(0, 0, 0, 0.3);
  z-index: 0;
}

/* 主题开关 */
.theme-switch {
  position: absolute;
  top: 20px;
  right: 20px;
  z-index: 2;
}

/* 登录框：玻璃拟态效果 */
.login-box {
  position: relative;
  z-index: 1;
  width: 400px;
  padding: 40px;
  background: rgba(255, 255, 255, 0.85); /* 白天模式：半透明白 */
  backdrop-filter: blur(10px);
  border-radius: 16px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.1);
  transition: all 0.3s;
}

/* 黑夜模式下的登录框 */
html.dark .login-box {
  background: rgba(0, 0, 0, 0.6); /* 黑夜模式：半透明黑 */
  color: #fff;
  border: 1px solid rgba(255, 255, 255, 0.1);
}

.header {
  text-align: center;
  margin-bottom: 30px;
}

.logo-avatar {
  background: var(--el-color-primary);
  margin-bottom: 10px;
  font-weight: bold;
}

.subtitle {
  color: #909399;
  font-size: 14px;
  margin-top: 5px;
}

.submit-btn {
  width: 100%;
  font-weight: bold;
  margin-top: 10px;
}

.toggle-mode {
  margin-top: 20px;
  text-align: center;
  font-size: 14px;
}

.toggle-mode a {
  color: var(--el-color-primary);
  cursor: pointer;
  font-weight: bold;
}
.toggle-mode a:hover {
  text-decoration: underline;
}
</style>