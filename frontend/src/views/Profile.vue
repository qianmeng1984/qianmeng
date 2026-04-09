<template>
  <div class="profile-container">
    <el-card class="profile-card" :class="{ 'wide-card': activeTab === 'feedback' }">
      <template #header>
        <div class="card-header">
          <div class="header-left">
            <span>个人中心</span>
          </div>
          <el-button link @click="router.back()">返回</el-button>
        </div>
      </template>

      <el-tabs v-model="activeTab" class="profile-tabs" @tab-click="handleTabClick">

        <el-tab-pane label="基本信息" name="info">
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
                <el-icon v-else class="avatar-uploader-icon"><Plus /></el-icon>
                <div class="upload-tip">点击更换头像</div>
              </el-upload>
            </div>

            <el-form :model="form" label-width="70px" class="info-form">
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
                <el-button type="primary" @click="handleSave" style="width: 100%; margin-top: 10px;">保存修改</el-button>
              </el-form-item>
            </el-form>
          </div>
        </el-tab-pane>

        <el-tab-pane label="问题反馈记录" name="feedback">
          <div class="feedback-list" v-loading="loadingFeedback">
            <el-empty v-if="dislikeList.length === 0" description="您还没有提交过问题反馈" :image-size="60"></el-empty>

            <div v-else class="timeline-wrapper">
              <el-timeline>
                <el-timeline-item
                    v-for="(item, index) in dislikeList"
                    :key="index"
                    :timestamp="formatDate(item.created_at || item.CreatedAt)"
                    placement="top"
                    :color="item.status === 1 ? '#67C23A' : '#F56C6C'"
                >
                  <el-card class="feedback-item-card clickable" shadow="hover" @click="openDetail(item)">
                    <div class="fb-row">
                      <div class="fb-left">
                        <div class="fb-status-badge">
                          <el-tag v-if="item.status === 1" type="success" effect="dark">已回复</el-tag>
                          <el-tag v-else type="info" effect="dark">待处理</el-tag>
                        </div>
                      </div>

                      <div class="fb-content">
                        <div class="fb-question">
                          <span class="label">Q:</span>
                          {{ truncate(item.ChatHistory?.question || item.ChatHistory?.Question, 40) }}
                        </div>
                        <div class="fb-reason-preview">
                          <span class="label">反馈:</span> {{ item.reason }}
                        </div>
                      </div>

                      <div class="fb-arrow">
                        <el-icon><ArrowRight /></el-icon>
                      </div>
                    </div>
                  </el-card>
                </el-timeline-item>
              </el-timeline>
            </div>
          </div>
        </el-tab-pane>
      </el-tabs>
    </el-card>

    <el-dialog v-model="detailVisible" title="反馈详情处理" width="600px" append-to-body>
      <div v-if="currentDetail" class="detail-container">

        <div class="detail-section">
          <div class="section-title">💬 对话上下文</div>
          <div class="chat-snapshot">
            <div class="snapshot-item user">
              <div class="role-tag">问</div>
              <div class="text">
                {{
                  currentDetail.ChatHistory?.question ||
                  currentDetail.ChatHistory?.Question ||
                  currentDetail.chat_history?.question ||
                  currentDetail.chat_history?.Question ||
                  '原问题已无法读取'
                }}
              </div>
            </div>
            <div class="snapshot-item ai">
              <div class="role-tag">答</div>
              <div class="text">
                {{
                  currentDetail.ChatHistory?.answer ||
                  currentDetail.ChatHistory?.Answer ||
                  currentDetail.chat_history?.answer ||
                  currentDetail.chat_history?.Answer ||
                  '原回答已无法读取'
                }}
              </div>
            </div>
          </div>
        </div>

        <div class="detail-section">
          <div class="section-title">📝 我的反馈意见</div>
          <div class="feedback-box">
            {{ currentDetail.reason }}
          </div>
        </div>

        <div class="detail-section" v-if="currentDetail.status === 1">
          <div class="section-title success">✅ 管理员回复</div>
          <div class="admin-reply-highlight">
            <el-icon class="reply-icon"><Service /></el-icon>
            <div class="reply-text">{{ currentDetail.admin_reply }}</div>
          </div>
        </div>
        <div class="detail-section" v-else>
          <el-alert title="管理员正在快马加鞭处理中..." type="info" show-icon :closable="false" />
        </div>

      </div>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="detailVisible = false">关闭</el-button>
          <el-button type="primary" @click="jumpToChat(currentDetail)">
            <el-icon style="margin-right:5px"><ChatLineRound /></el-icon> 跳转至该对话
          </el-button>
        </span>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { reactive, onMounted, computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { getUserInfo } from '@/api/user'
import request from '@/utils/request'
import { ElMessage } from 'element-plus'
import { Plus, Service, ArrowRight, ChatLineRound } from '@element-plus/icons-vue'

const router = useRouter()
const uploadActionUrl = 'http://localhost:8080/api/v1/upload/avatar'
const BASE_URL = 'http://localhost:8080'

// Tabs 控制
const activeTab = ref('info')
const loadingFeedback = ref(false)
const feedbackList = ref([])
const detailVisible = ref(false)
const currentDetail = ref(null)

// 过滤：只显示点踩 (type == 2) 的记录
const dislikeList = computed(() => {
  return feedbackList.value.filter(item => item.type === 2)
})

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
  if (!path) return `${BASE_URL}/uploads/avatars/default.png`
  let url = path
  if (!path.startsWith('http')) {
    url = BASE_URL + path
  }
  return `${url}?t=${form.timestamp}`
}

const formatDate = (dateStr) => {
  if (!dateStr) return ''
  return new Date(dateStr).toLocaleString('zh-CN', { hour12: false })
}

const truncate = (str, len) => {
  if (!str) return ''
  if (str.length > len) return str.substring(0, len) + '...'
  return str
}

const handleTabClick = (tab) => {
  if (tab.paneName === 'feedback') {
    fetchMyFeedbacks()
  }
}

const fetchMyFeedbacks = async () => {
  loadingFeedback.value = true
  try {
    const res = await request({ url: '/feedback/my', method: 'get' })
    feedbackList.value = res || []
  } catch (e) { console.error(e) } finally { loadingFeedback.value = false }
}

// 打开详情
const openDetail = (item) => {
  currentDetail.value = item
  detailVisible.value = true
}

// 跳转到聊天页面
const jumpToChat = (item) => {
  // 1. 获取关联的 history 对象
  const history = item.ChatHistory || item.chat_history

  if (!history) {
    ElMessage.warning('无法跳转：找不到关联的对话记录')
    return
  }

  // 2. 获取 ID (兼容大小写)
  const convId = history.conversation_id || history.ConversationID || history.ID || history.id

  if (convId) {
    console.log("正在跳转到会话 ID:", convId)

    // 🚩【核心修改】路径改为 '/' (根路径)，而不是 '/chat'
    router.push({ path: '/', query: { conversation_id: convId } })
  } else {
    ElMessage.warning('数据异常：无法获取会话ID')
  }
}

onMounted(async () => {
  try {
    const res = await getUserInfo()
    form.username = res.Username || res.username
    form.nickname = res.Nickname || res.nickname
    form.avatar = res.Avatar || res.avatar
  } catch (e) { console.error(e) }
})

const handleAvatarSuccess = (res) => {
  if (res.url) {
    form.avatar = res.url
    form.timestamp = Date.now()
    ElMessage.success('头像上传成功')
  }
}

const handleSave = async () => {
  try {
    await request({
      url: '/user/update',
      method: 'post',
      data: { nickname: form.nickname, avatar: form.avatar, password: form.password }
    })
    ElMessage.success('保存成功')
    if (form.password) {
      localStorage.clear()
      router.push('/login')
    }
  } catch (e) { console.error(e) }
}
</script>

<style scoped>
.profile-container {
  display: flex;
  justify-content: center;
  padding-top: 50px;
  min-height: 100vh;
}

/* 核心动画：卡片宽度过渡 */
.profile-card {
  width: 500px;
  height: fit-content;
  border-radius: 12px;
  transition: width 0.4s cubic-bezier(0.25, 0.8, 0.25, 1); /* 平滑弹跳效果 */
}
/* 激活态宽度 */
.wide-card {
  width: 900px;
}

.card-header {
  display: flex; justify-content: space-between; align-items: center; font-size: 16px; font-weight: bold;
}
.avatar-section {
  display: flex; flex-direction: column; align-items: center; margin-bottom: 25px; margin-top: 10px;
}
.avatar {
  width: 100px; height: 100px; border-radius: 50%; object-fit: cover; border: 3px solid #f0f2f5;
}
.upload-tip { font-size: 12px; color: #999; margin-top: 8px; }
.profile-tabs { min-height: 400px; }

/* 反馈列表样式 */
.feedback-list { padding: 10px 20px; }
.timeline-wrapper { padding-top: 10px; }

.feedback-item-card {
  cursor: pointer;
  border: 1px solid #eee;
  transition: all 0.2s;
}
.feedback-item-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0,0,0,0.1);
  border-color: #409EFF;
}

.fb-row { display: flex; align-items: center; gap: 15px; }
.fb-status-badge { min-width: 60px; }
.fb-content { flex: 1; overflow: hidden; }
.fb-arrow { color: #ccc; }

.fb-question { font-weight: bold; color: #333; margin-bottom: 4px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.fb-question .label { color: #409EFF; margin-right: 4px; }
.fb-reason-preview { font-size: 13px; color: #666; }
.fb-reason-preview .label { color: #E6A23C; margin-right: 4px; }

/* 详情弹窗样式 */
.detail-container { padding: 0 10px; }
.detail-section { margin-bottom: 25px; }
.section-title { font-size: 14px; font-weight: bold; color: #303133; margin-bottom: 10px; padding-left: 8px; border-left: 4px solid #409EFF; }
.section-title.success { border-left-color: #67C23A; color: #67C23A; }

.chat-snapshot { background: #f5f7fa; padding: 15px; border-radius: 8px; border: 1px dashed #dcdfe6; }
.snapshot-item { display: flex; margin-bottom: 10px; gap: 10px; }
.snapshot-item:last-child { margin-bottom: 0; }
.snapshot-item .role-tag { flex-shrink: 0; width: 24px; height: 24px; border-radius: 4px; text-align: center; line-height: 24px; font-size: 12px; color: #fff; }
.snapshot-item.user .role-tag { background: #409EFF; }
.snapshot-item.ai .role-tag { background: #67C23A; }
.snapshot-item .text { font-size: 14px; color: #606266; line-height: 1.5; }

.feedback-box { background: #fdf6ec; padding: 12px; border-radius: 6px; color: #e6a23c; font-size: 14px; }

.admin-reply-highlight { background: #f0f9eb; padding: 15px; border-radius: 8px; border: 1px solid #e1f3d8; display: flex; gap: 12px; }
.reply-icon { font-size: 20px; color: #67C23A; margin-top: 2px; }
.reply-text { color: #333; font-size: 15px; line-height: 1.6; }

/* 黑夜模式适配 */
html.dark .profile-card { background: rgba(30, 30, 30, 0.95); border-color: rgba(255,255,255,0.1); }
html.dark .feedback-item-card { background: rgba(255,255,255,0.05); border-color: rgba(255,255,255,0.1); }
html.dark .fb-question { color: #eee; }
html.dark .fb-reason-preview { color: #aaa; }
html.dark .chat-snapshot { background: rgba(0,0,0,0.3); border-color: rgba(255,255,255,0.1); }
html.dark .snapshot-item .text { color: #ccc; }
html.dark .feedback-box { background: rgba(230, 162, 60, 0.1); color: #E6A23C; }
html.dark .admin-reply-highlight { background: rgba(103, 194, 58, 0.1); border-color: rgba(103, 194, 58, 0.2); }
html.dark .reply-text { color: #e0e0e0; }
html.dark .el-dialog { background: #1e1e1e; }
html.dark .el-dialog__title { color: #eee; }
</style>