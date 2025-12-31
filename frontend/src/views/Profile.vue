<template>
  <div class="profile-container">
    <el-card class="profile-card">
      <template #header>
        <div class="card-header">
          <span>👤 个人中心</span>
          <el-button link @click="router.back()">返回</el-button>
        </div>
      </template>

      <div class="content">
        <div class="avatar-section">
          <el-upload
              class="avatar-uploader"
              :action="uploadActionUrl"
              :headers="headers"
              :show-file-list="false"
              :on-success="handleAvatarSuccess"
              name="file"
          >
            <img v-if="form.avatar" :src="getFullUrl(form.avatar)" class="avatar" />
            <el-icon v-else class="avatar-uploader-icon">+</el-icon>
            <div class="upload-tip">点击更换头像</div>
          </el-upload>
        </div>

        <el-form :model="form" label-width="80px">
          <el-form-item label="账号">
            <el-input v-model="form.username" disabled />
          </el-form-item>
          <el-form-item label="昵称">
            <el-input v-model="form.nickname" />
          </el-form-item>
          <el-form-item label="新密码">
            <el-input v-model="form.password" type="password" show-password placeholder="不改请留空" autocomplete="new-password"/>
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="handleSave">保存修改</el-button>
          </el-form-item>
        </el-form>
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { reactive, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { getUserInfo } from '@/api/user'
import request from '@/utils/request'
import { ElMessage } from 'element-plus'

const router = useRouter()
const uploadActionUrl = 'http://localhost:8080/api/v1/upload/avatar'
const BASE_URL = 'http://localhost:8080'

const form = reactive({
  username: '',
  nickname: '',
  avatar: '',
  password: '',
  timestamp: Date.now()
})

const headers = computed(() => ({
  Authorization: `Bearer ${localStorage.getItem('token')}`
}))

// 获取完整头像路径
const getFullUrl = (path) => {
  if (!path) return '/uploads/avatars/default.png'
  let url = path
  if (!path.startsWith('http')) {
    url = BASE_URL + path
  }
  return `${url}?t=${form.timestamp}`
}

onMounted(async () => {
  try {
    const res = await getUserInfo()
    form.username = res.Username
    form.nickname = res.Nickname
    form.avatar = res.Avatar
  } catch (e) {
    console.error(e)
  }
})

const handleAvatarSuccess = (res) => {
  if (res.url) {
    form.avatar = res.url
    form.timestamp = Date.now()
    ElMessage.success('头像上传成功，记得点保存')
  }
}

const handleSave = async () => {
  try {
    await request({
      url: '/user/update',
      method: 'post',
      data: {
        nickname: form.nickname,
        avatar: form.avatar,
        password: form.password
      }
    })
    ElMessage.success('保存成功')
    if (form.password) {
      ElMessage.warning('密码已修改，请重新登录')
      localStorage.clear()
      router.push('/login')
    }
  } catch (e) {
    console.error(e)
  }
}
</script>

<style scoped>
.profile-container {
  display: flex;
  justify-content: center;
  padding-top: 50px;
  /* background: #f4f4f7;  <-- 【核心修改】去掉了这行背景色，让星空透出来 */
  min-height: 100vh;
}

.profile-card {
  width: 500px;
  height: fit-content;
}

.avatar-section {
  display: flex;
  flex-direction: column;
  align-items: center;
  margin-bottom: 20px;
}

.avatar {
  width: 100px;
  height: 100px;
  border-radius: 50%;
  object-fit: cover;
  border: 2px solid #eee;
}

.upload-tip {
  font-size: 12px;
  color: #999;
  margin-top: 5px;
}

/* App.vue 里的全局样式会自动处理 .el-card 在黑夜模式下的透明度和毛玻璃效果
   所以这里不需要额外写黑夜模式的 CSS
*/
</style>