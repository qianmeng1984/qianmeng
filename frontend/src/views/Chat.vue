<template>
  <div class="chat-container">
    <aside class="sidebar" :class="{ 'collapsed': isSidebarCollapsed }">
      <div class="sidebar-header">
        <el-button
            v-if="!isSidebarCollapsed"
            type="primary"
            class="new-chat-btn"
            @click="startNewChat"
        >
          <el-icon><Plus /></el-icon> 新建对话
        </el-button>
        <el-button link :icon="isSidebarCollapsed ? Expand : Fold" @click="isSidebarCollapsed = !isSidebarCollapsed" />
      </div>

      <div class="history-list">
        <div
            v-for="(item, index) in conversationList"
            :key="index"
            class="history-item"
            :class="{ 'active': currentConversationId === item.ID }"
            @click="loadConversation(item.ID)"
        >
          <div class="history-content">
            <el-icon><ChatDotRound /></el-icon>
            <span class="history-title">{{ item.title || item.Title || '无标题会话' }}</span>
          </div>

          <div class="history-actions" v-if="!isSidebarCollapsed" @click.stop>
            <el-dropdown trigger="click" @command="(cmd) => handleHistoryCommand(cmd, item)">
              <el-icon class="more-btn"><MoreFilled /></el-icon>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="rename"><el-icon><Edit /></el-icon>重命名</el-dropdown-item>
                  <el-dropdown-item command="delete" style="color: #F56C6C"><el-icon><Delete /></el-icon>删除</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>
        </div>
        <el-empty v-if="conversationList.length === 0" description="暂无历史" :image-size="50" />
      </div>

      <div class="sidebar-footer">
        <el-button v-if="!isSidebarCollapsed" type="danger" link @click="handleLogout">退出登录</el-button>
      </div>
    </aside>

    <main class="main-chat">
      <header class="chat-header">
        <h2>RAG 智能助手</h2>

        <div class="header-right">
          <el-button circle :icon="isDark ? Moon : Sunny" @click="toggleTheme" style="margin-right: 15px;" />

          <el-button
              v-if="role === 'admin'"
              type="danger"
              plain
              size="small"
              @click="router.push('/admin')"
              style="margin-right: 10px"
          >
            <el-icon style="margin-right: 5px"><Setting /></el-icon> 后台管理
          </el-button>

          <el-button type="primary" plain size="small" @click="router.push('/knowledge')" style="margin-right: 15px">
            <el-icon style="margin-right: 5px"><Upload /></el-icon> 知识库管理
          </el-button>

          <el-dropdown trigger="click" @command="handleCommand">
            <div class="user-info-trigger">
              <span class="username">{{ currentUser.nickname || '用户' }}</span>
              <el-avatar :size="40" :src="getFullUrl(currentUser.avatar)">
                {{ currentUser.nickname ? currentUser.nickname.charAt(0) : 'U' }}
              </el-avatar>
            </div>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="profile"><el-icon><User /></el-icon> 个人中心</el-dropdown-item>
                <el-dropdown-item divided command="logout" style="color: #f56c6c"><el-icon><SwitchButton /></el-icon> 退出登录</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </header>

      <div class="messages" ref="msgContainer">
        <div v-if="messages.length === 0 && !loading" class="welcome-box">
          <h3>👋 欢迎回来！</h3>
          <p>我是你的智能知识库助手，你可以问我任何问题。</p>
        </div>

        <div v-for="(msg, index) in messages" :key="index" class="message-row" :class="msg.role">
          <div class="avatar">
            <el-avatar v-if="msg.role === 'assistant'" :icon="Service" style="background: #67C23A" />
            <el-avatar v-else :src="getFullUrl(currentUser.avatar)" style="background: #409EFF">
              {{ currentUser.nickname ? currentUser.nickname.charAt(0) : 'U' }}
            </el-avatar>
          </div>
          <div class="bubble">
            <div class="content">{{ msg.content }}</div>
          </div>
        </div>

        <div v-if="loading" class="message-row assistant">
          <div class="avatar"><el-avatar :icon="Service" style="background: #67C23A" /></div>
          <div class="bubble loading">正在思考...</div>
        </div>
      </div>

      <div class="input-area">
        <div class="input-box">
          <el-input
              v-model="input"
              type="textarea"
              :autosize="{ minRows: 1, maxRows: 4 }"
              placeholder="请输入您的问题..."
              @keydown.enter.prevent="handleSend"
          />
          <el-button type="primary" :icon="Position" @click="handleSend" :loading="loading" class="send-btn" />
        </div>
      </div>
    </main>
  </div>
</template>

<script setup>
import { ref, onMounted, nextTick } from 'vue'
import { useRouter } from 'vue-router'
// 👇 引入新增的 API
import { chat, getConversationList, getConversationMessages, renameConversation, deleteConversation } from '@/api/chat'
import { getUserInfo } from '@/api/user'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Fold, Expand, ChatDotRound, User, Service, Position, Upload, SwitchButton, Plus, Setting, Moon, Sunny,
  MoreFilled, Edit, Delete // 👈 引入新图标
} from '@element-plus/icons-vue'

const router = useRouter()
const role = ref('')
const isDark = ref(false)

const isSidebarCollapsed = ref(false)
const input = ref('')
const loading = ref(false)
const conversationList = ref([])
const messages = ref([])
const currentConversationId = ref(0)
const msgContainer = ref(null)
const currentUser = ref({ nickname: '', avatar: '', role: 0 })
const BASE_URL = 'http://localhost:8080'

const getFullUrl = (path) => {
  if (!path) return `${BASE_URL}/uploads/avatars/default.png`
  let url = path.startsWith('http') ? path : BASE_URL + path
  return `${url}?t=${new Date().getTime()}`
}

const scrollToBottom = () => {
  nextTick(() => {
    if (msgContainer.value) {
      msgContainer.value.scrollTop = msgContainer.value.scrollHeight
    }
  })
}

const toggleTheme = () => {
  isDark.value = !isDark.value
  if (isDark.value) {
    document.documentElement.classList.add('dark')
    localStorage.setItem('theme', 'dark')
  } else {
    document.documentElement.classList.remove('dark')
    localStorage.setItem('theme', 'light')
  }
}

onMounted(() => {
  fetchUserInfo()
  fetchConversations()

  const savedTheme = localStorage.getItem('theme')
  if (savedTheme === 'dark') {
    isDark.value = true
    document.documentElement.classList.add('dark')
  } else {
    isDark.value = false
    document.documentElement.classList.remove('dark')
  }
})

const fetchConversations = async () => {
  try {
    const res = await getConversationList()
    conversationList.value = res || []
  } catch (e) {}
}

const loadConversation = async (id) => {
  if (currentConversationId.value === id) return
  currentConversationId.value = id
  messages.value = []
  try {
    const res = await getConversationMessages(id)
    if (res && res.length > 0) {
      res.forEach(item => {
        messages.value.push({ role: 'user', content: item.question || item.Question })
        messages.value.push({ role: 'assistant', content: item.answer || item.Answer })
      })
    }
    scrollToBottom()
  } catch (e) {}
}

const startNewChat = () => {
  currentConversationId.value = 0
  messages.value = []
}

// 👇👇👇 新增：处理会话菜单操作 👇👇👇
const handleHistoryCommand = async (cmd, item) => {
  if (cmd === 'rename') {
    ElMessageBox.prompt('请输入新的会话标题', '重命名', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      inputValue: item.title || item.Title,
    }).then(async ({ value }) => {
      if (!value) return
      await renameConversation(item.ID, value)
      ElMessage.success('修改成功')
      fetchConversations() // 刷新列表
    }).catch(() => {})
  } else if (cmd === 'delete') {
    ElMessageBox.confirm(
        '确定删除该会话及其所有记录吗？此操作不可逆。',
        '警告',
        { confirmButtonText: '删除', cancelButtonText: '取消', type: 'warning' }
    ).then(async () => {
      await deleteConversation(item.ID)
      ElMessage.success('删除成功')

      // 如果删除的是当前选中的，重置到新对话
      if (currentConversationId.value === item.ID) {
        startNewChat()
      }
      fetchConversations()
    }).catch(() => {})
  }
}

const handleSend = async () => {
  if (!input.value.trim() || loading.value) return
  const question = input.value
  input.value = ''
  messages.value.push({ role: 'user', content: question })
  scrollToBottom()
  loading.value = true
  try {
    const res = await chat(question, currentConversationId.value)
    messages.value.push({ role: 'assistant', content: res.answer })
    if (currentConversationId.value === 0 && res.conversation_id) {
      currentConversationId.value = res.conversation_id
      await fetchConversations()
    }
  } catch (error) {
    messages.value.push({ role: 'assistant', content: '抱歉，出错了。' })
  } finally {
    loading.value = false
    scrollToBottom()
  }
}

const fetchUserInfo = async () => {
  try {
    const res = await getUserInfo()
    currentUser.value = {
      nickname: res.Nickname || res.nickname || '用户',
      avatar: res.Avatar || res.avatar || '',
      role: res.Role || res.role || 0
    }
    role.value = (res.Role === 1 || res.role === 1) ? 'admin' : 'user'
  } catch (e) {}
}

const handleCommand = (cmd) => {
  if (cmd === 'profile') router.push('/profile')
  else if (cmd === 'logout') handleLogout()
}

const handleLogout = () => {
  localStorage.clear()
  document.documentElement.classList.remove('dark')
  router.push('/login')
}
</script>

<style scoped>
/* 默认(亮色)模式下的样式 */
.chat-container { display: flex; height: 100vh; background-color: #f4f4f7; transition: background-color 0.3s; }
.sidebar { width: 260px; background: #fff; border-right: 1px solid #e0e0e0; display: flex; flex-direction: column; transition: all 0.3s; }
.sidebar.collapsed { width: 60px; }
.sidebar-header { padding: 15px; display: flex; gap: 10px; align-items: center; border-bottom: 1px solid #eee; }
.new-chat-btn { flex: 1; font-weight: bold; }
.sidebar.collapsed .new-chat-btn { display: none; }
.history-list { flex: 1; overflow-y: auto; padding: 10px; }

/* 👇👇👇 修改：history-item 改为 flex 布局以支持右侧图标 👇👇👇 */
.history-item {
  display: flex;
  align-items: center;
  justify-content: space-between; /* 两端对齐 */
  padding: 12px;
  cursor: pointer;
  border-radius: 8px;
  color: #666;
  font-size: 14px;
  margin-bottom: 5px;
  transition: all 0.2s;
  group: true; /* 允许子元素感知 hover */
}
.history-item:hover { background: #f0f2f5; color: #333; }
.history-item.active { background: #e6f7ff; color: #409EFF; font-weight: bold; }

.history-content { display: flex; align-items: center; gap: 10px; overflow: hidden; }
.history-title { white-space: nowrap; overflow: hidden; text-overflow: ellipsis; max-width: 140px; }
.sidebar.collapsed .history-title { display: none; }

/* 👇👇👇 新增：操作按钮样式 👇👇👇 */
.history-actions {
  display: none; /* 默认隐藏 */
  padding: 2px;
}
.history-item:hover .history-actions { display: block; /* 悬停时显示 */ }
.more-btn { font-size: 16px; color: #999; transform: rotate(90deg); }
.more-btn:hover { color: #409EFF; }


.sidebar-footer { padding: 20px; border-top: 1px solid #eee; text-align: center; }
.main-chat { flex: 1; display: flex; flex-direction: column; position: relative; background: #f4f4f7; transition: background 0.3s; }
.chat-header { height: 60px; background: #fff; border-bottom: 1px solid #eee; display: flex; align-items: center; justify-content: space-between; padding: 0 20px; transition: background 0.3s, border 0.3s; }
.header-right { display: flex; align-items: center; }
.user-info-trigger { display: flex; align-items: center; gap: 10px; cursor: pointer; padding: 5px 10px; border-radius: 20px; transition: background 0.3s; }
.user-info-trigger:hover { background: rgba(0,0,0,0.05); }
.username { font-size: 14px; font-weight: 500; color: #333; transition: color 0.3s; }
.messages { flex: 1; padding: 20px; overflow-y: auto; display: flex; flex-direction: column; gap: 20px; }
.welcome-box { text-align: center; margin-top: 100px; color: #999; }
.message-row { display: flex; gap: 15px; max-width: 80%; }
.message-row.user { align-self: flex-end; flex-direction: row-reverse; }
.message-row .bubble { background: #fff; padding: 12px 16px; border-radius: 12px; box-shadow: 0 2px 8px rgba(0,0,0,0.05); line-height: 1.6; font-size: 15px; white-space: pre-wrap; transition: background 0.3s, color 0.3s; }
.message-row.user .bubble { background: #409EFF; color: #fff; border-radius: 12px 12px 0 12px; }
.message-row.assistant .bubble { border-radius: 0 12px 12px 12px; }
.input-area { background: #fff; padding: 20px; border-top: 1px solid #eee; transition: background 0.3s, border 0.3s; }
.input-box { max-width: 800px; margin: 0 auto; display: flex; gap: 10px; position: relative; }
.send-btn { position: absolute; right: 10px; bottom: 5px; }

/* ========================================= */
/* 🌌 黑夜模式适配 */
/* ========================================= */
html.dark .chat-container { background-color: transparent; }
html.dark .main-chat { background-color: transparent; }

html.dark .sidebar {
  background: rgba(20, 20, 20, 0.6);
  backdrop-filter: blur(10px);
  border-right: 1px solid rgba(255, 255, 255, 0.1);
}
html.dark .sidebar-header { border-bottom: 1px solid rgba(255, 255, 255, 0.1); color: #ccc; }
html.dark .sidebar-footer { border-top: 1px solid rgba(255, 255, 255, 0.1); }

html.dark .history-item { color: #a0a0a0; }
html.dark .history-item:hover { background: rgba(255, 255, 255, 0.1); color: #fff; }
html.dark .history-item.active { background: rgba(64, 158, 255, 0.2); color: #409EFF; }

html.dark .chat-header {
  background: rgba(20, 20, 20, 0.6);
  backdrop-filter: blur(10px);
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}
html.dark .chat-header h2 { color: #e0e0e0; text-shadow: 0 0 5px rgba(0,0,0,0.5); }
html.dark .username { color: #ccc; }
html.dark .user-info-trigger:hover { background: rgba(255,255,255,0.1); }

html.dark .input-area {
  background: rgba(20, 20, 20, 0.6);
  backdrop-filter: blur(10px);
  border-top: 1px solid rgba(255, 255, 255, 0.1);
}

html.dark .message-row.assistant .bubble {
  background: rgba(45, 45, 45, 0.85);
  color: #e0e0e0;
  box-shadow: none;
  border: 1px solid rgba(255, 255, 255, 0.1);
}

html.dark .input-box .el-textarea__inner {
  background-color: rgba(45, 45, 45, 0.5) !important;
  color: #fff !important;
  box-shadow: none !important;
  border: 1px solid rgba(255, 255, 255, 0.2);
}
html.dark .input-box .el-textarea__inner:focus {
  border-color: #409EFF;
}
html.dark .welcome-box { color: #bbb; text-shadow: 0 0 2px #000; }
</style>