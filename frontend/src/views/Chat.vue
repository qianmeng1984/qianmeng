<template>
  <div class="chat-container">
    <aside class="sidebar" :class="{ 'collapsed': isSidebarCollapsed }">
      <div class="sidebar-header">
        <el-button
            v-if="!isSidebarCollapsed"
            type="primary"
            class="new-chat-btn"
            size="large"
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
          <div class="history-content" :title="item.title || item.Title">
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
        <el-empty v-if="conversationList.length === 0" description="暂无历史" :image-size="60" />
      </div>

      <div class="sidebar-footer">
        <el-button v-if="!isSidebarCollapsed" type="danger" link style="font-size: 15px" @click="handleLogout">退出登录</el-button>
      </div>
    </aside>

    <main class="main-chat">
      <header class="chat-header">
        <h2>RAG 智能助手 (HIST版)</h2>

        <div class="header-right">
          <el-button circle size="large" :icon="isDark ? Moon : Sunny" @click="toggleTheme" style="margin-right: 15px;" />

          <el-button v-if="role === 'admin'" type="danger" plain size="large" @click="router.push('/admin')" style="margin-right: 10px">
            <el-icon style="margin-right: 5px"><Setting /></el-icon> 后台管理
          </el-button>

          <el-button v-if="role === 'admin'" type="primary" plain size="large" @click="router.push('/knowledge')" style="margin-right: 20px">
            <el-icon style="margin-right: 5px"><Upload /></el-icon> 知识库管理
          </el-button>

          <el-dropdown trigger="click" @command="handleCommand">
            <div class="user-info-trigger">
              <span class="username">{{ currentUser.nickname || '用户' }}</span>
              <el-avatar :size="44" :src="getFullUrl(currentUser.avatar)">
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

      <div class="announcement-banner" v-if="sysAnnouncement">
        <div class="banner-icon"><el-icon><Bell /></el-icon></div>
        <div class="marquee-container">
          <div class="marquee-content">
            <strong>【{{ sysAnnouncement.title }}】</strong> {{ sysAnnouncement.content }}
          </div>
        </div>
      </div>

      <div class="messages" ref="msgContainer">
        <div v-if="messages.length === 0 && !loading" class="welcome-box">
          <h3>👋 欢迎回来！</h3>
          <p>我是你的智能知识库助手，你可以问我关于<b>河南科技学院</b>的任何问题。</p>

          <div class="prompt-cards" v-if="activePrompts.length > 0">
            <div
                class="prompt-card"
                v-for="prompt in activePrompts"
                :key="prompt.ID"
                @click="handleSend(prompt.content)"
            >
              <h4><el-icon><ChatLineSquare /></el-icon> {{ prompt.title }}</h4>
              <p>{{ prompt.content }}</p>
            </div>
          </div>
        </div>

        <div v-for="(msg, index) in messages" :key="index" class="message-row" :class="msg.role">
          <div class="avatar">
            <el-avatar v-if="msg.role === 'assistant'" :size="40" :icon="Service" style="background: #67C23A" />
            <el-avatar v-else :size="40" :src="getFullUrl(currentUser.avatar)" style="background: #409EFF">
              {{ currentUser.nickname ? currentUser.nickname.charAt(0) : 'U' }}
            </el-avatar>
          </div>

          <div class="bubble-container">
            <div v-if="msg.role === 'assistant' && msg.thinking" class="thinking-panel">
              <el-collapse :model-value="['1']">
                <el-collapse-item name="1">
                  <template #title>
                    <el-icon class="is-loading" v-if="loading && index === messages.length -1" style="margin-right: 8px"><Loading /></el-icon>
                    <el-icon v-else style="margin-right: 8px; color: #67C23A; font-size: 16px;"><Checked /></el-icon>
                    <span class="thinking-title">RAG 智能分析引擎 (Deep Search)</span>
                  </template>

                  <div class="think-content">
                    <div class="think-row">
                      <span class="label">🔍 意图重写:</span>
                      <span class="value strong">{{ msg.thinking.rewritten_query }}</span>
                    </div>
                    <div class="think-row">
                      <span class="label">🚀 向量去噪:</span>
                      <span class="value code">{{ msg.thinking.vector_query }}</span>
                    </div>
                    <div class="think-row">
                      <span class="label">💥 关键词:</span>
                      <div class="tags">
                        <el-tag v-for="kw in msg.thinking.keywords" :key="kw" size="default" effect="plain">{{ kw }}</el-tag>
                      </div>
                    </div>

                    <div class="think-row" v-if="msg.thinking.sources && msg.thinking.sources.length > 0">
                      <span class="label">📚 引用溯源:</span>
                      <ul class="source-list">
                        <li
                            v-for="(src, idx) in msg.thinking.sources"
                            :key="idx"
                            @click="openSourceDetail(src, msg.thinking.keywords)"
                            class="source-item"
                            :title="src"
                        >
                          <el-tag size="small" :type="parseSource(src).type" effect="dark" class="mini-tag">
                            {{ parseSource(src).score }}
                          </el-tag>
                          <span class="source-text">
                            <span class="fname">{{ parseSource(src).fileName }}</span>
                            <span class="sep">|</span>
                            <span class="preview" v-html="highlightPreview(parseSource(src).shortContent, msg.thinking.keywords)"></span>
                          </span>
                        </li>
                      </ul>
                    </div>
                    <div class="think-footer">
                      🧠 模型: Doubao-Pro-32k
                    </div>
                  </div>
                </el-collapse-item>
              </el-collapse>
            </div>

            <div class="bubble">
              <div class="content" v-if="msg.content">{{ msg.content }}</div>
              <div class="content" v-else-if="loading && index === messages.length - 1" style="color: #999; font-style: italic;">
                <span class="typing-cursor">正在思考...</span>
              </div>
            </div>

            <div v-if="msg.role === 'assistant' && !loading" class="feedback-actions">
              <el-tooltip content="复制内容" placement="top" :show-after="500">
                <el-icon class="action-icon copy-btn" @click="copyToClipboard(msg.content)"><DocumentCopy /></el-icon>
              </el-tooltip>

              <el-tooltip v-if="index === messages.length - 1" content="重新回答" placement="top" :show-after="500">
                <el-icon class="action-icon retry-btn" @click="handleRegenerate()"><RefreshRight /></el-icon>
              </el-tooltip>

              <div class="divider"></div>

              <el-tooltip content="回答很有帮助" placement="top" :show-after="500">
                <el-icon class="action-icon like-btn" :class="{ active: msg.feedbackType === 1, disabled: !msg.id }" @click="msg.id && handleLike(msg)">
                  <SuccessFilled v-if="msg.feedbackType === 1" />
                  <CircleCheck v-else />
                </el-icon>
              </el-tooltip>

              <el-tooltip content="回答有误/不满意" placement="top" :show-after="500">
                <el-icon class="action-icon dislike-btn" :class="{ active: msg.feedbackType === 2, disabled: !msg.id }" @click="msg.id && handleDislike(msg)">
                  <CircleCloseFilled v-if="msg.feedbackType === 2" />
                  <CircleClose v-else />
                </el-icon>
              </el-tooltip>
            </div>
          </div>
        </div>
      </div>

      <div class="input-area">
        <div class="input-box">
          <el-input
              v-model="input"
              type="textarea"
              :autosize="{ minRows: 2, maxRows: 5 }"
              placeholder="请输入您的问题..."
              @keydown.enter.prevent="handleSend()"
          />
          <el-button type="primary" size="large" :icon="Position" @click="handleSend()" :loading="loading" class="send-btn" />
        </div>
      </div>

      <el-drawer v-model="drawerVisible" title="📄 文档切片溯源" :direction="'rtl'" size="45%" class="source-drawer">
        <div class="drawer-content">
          <el-alert
              title="检索命中详情"
              type="info"
              :closable="false"
              description="以下是系统根据您的关键词检索到的原始文档切片内容。"
              show-icon
              style="margin-bottom: 20px"
          />
          <div class="meta-info">
            <el-tag effect="dark" size="large" style="font-size: 15px; padding: 0 15px; height: 35px; line-height: 35px;">{{ currentSource.fileName }}</el-tag>
          </div>
          <div class="raw-content-box">
            <p v-html="currentSource.highlightedContent"></p>
          </div>
        </div>
      </el-drawer>

      <el-dialog v-model="feedbackDialogVisible" title="提交反馈" width="35%" class="large-dialog">
        <el-input
            v-model="feedbackReason"
            type="textarea"
            :rows="5"
            style="font-size: 16px;"
            placeholder="请告诉我们哪里回答得不好（例如：答非所问、信息过时、遗漏关键点等），我们将努力改进..."
        />
        <template #footer>
          <span class="dialog-footer">
            <el-button size="large" @click="feedbackDialogVisible = false">取消</el-button>
            <el-button size="large" type="primary" @click="submitDislike">提交反馈</el-button>
          </span>
        </template>
      </el-dialog>

    </main>
  </div>
</template>

<script setup>
import { ref, onMounted, nextTick } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { getConversationList, getConversationMessages, renameConversation, deleteConversation } from '@/api/chat'
import { getUserInfo } from '@/api/user'
import request from '@/utils/request'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Fold, Expand, ChatDotRound, User, Service, Position, Upload, SwitchButton, Plus, Setting, Moon, Sunny,
  MoreFilled, Edit, Delete, Loading, Checked, Bell, ChatLineSquare,
  CircleCheck, CircleClose, DocumentCopy, RefreshRight, SuccessFilled, CircleCloseFilled
} from '@element-plus/icons-vue'

const router = useRouter()
const route = useRoute()
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

const drawerVisible = ref(false)
const currentSource = ref({ fileName: '', highlightedContent: '' })
const feedbackDialogVisible = ref(false)
const feedbackReason = ref('')
const currentFeedbackMsg = ref(null)

const sysAnnouncement = ref(null)
// 🚀 新增：存储快捷指令的变量
const activePrompts = ref([])

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

const fetchLatestAnnouncement = async () => {
  try {
    const res = await request.get('/announcement/latest')
    if (res && res.title) {
      sysAnnouncement.value = res
    }
  } catch (e) { console.error("无法获取公告", e) }
}

// 🚀 新增：获取启用的快捷指令
const fetchActivePrompts = async () => {
  try {
    const res = await request.get('/prompt/active')
    activePrompts.value = res || []
  } catch (e) { console.error("无法获取指令", e) }
}

onMounted(async () => {
  fetchUserInfo()
  await fetchConversations()
  fetchLatestAnnouncement()
  fetchActivePrompts() // 页面加载时请求快捷指令

  const savedTheme = localStorage.getItem('theme')
  if (savedTheme === 'dark') {
    isDark.value = true
    document.documentElement.classList.add('dark')
  } else {
    isDark.value = false
    document.documentElement.classList.remove('dark')
  }

  if (route.query.conversation_id) {
    const targetId = parseInt(route.query.conversation_id)
    if (targetId) {
      setTimeout(() => {
        loadConversation(targetId)
      }, 100)
    }
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

        let thinkingObj = null
        const logStr = item.thinking_log || item.ThinkingLog
        if (logStr && logStr.length > 0) {
          try {
            thinkingObj = JSON.parse(logStr)
          } catch (e) { console.error("解析失败", e) }
        }

        messages.value.push({
          id: item.ID || item.id,
          role: 'assistant',
          content: item.answer || item.Answer,
          thinking: thinkingObj,
          feedbackType: 0
        })
      })
    }
    scrollToBottom()
  } catch (e) { console.error(e) }
}

const startNewChat = () => {
  currentConversationId.value = 0
  messages.value = []
}

const handleHistoryCommand = async (cmd, item) => {
  if (cmd === 'rename') {
    ElMessageBox.prompt('请输入新的会话标题', '重命名', {
      confirmButtonText: '确定', cancelButtonText: '取消', inputValue: item.title || item.Title,
    }).then(async ({value}) => {
      if (!value) return
      await renameConversation(item.ID, value)
      ElMessage.success('修改成功')
      fetchConversations()
    }).catch(() => {})
  } else if (cmd === 'delete') {
    ElMessageBox.confirm('确定删除该会话及其所有记录吗？', '警告', {
      confirmButtonText: '删除', cancelButtonText: '取消', type: 'warning'
    }).then(async () => {
      await deleteConversation(item.ID)
      ElMessage.success('删除成功')
      if (currentConversationId.value === item.ID) startNewChat()
      fetchConversations()
    }).catch(() => {})
  }
}

const openSourceDetail = (src, keywords) => {
  const parts = parseSource(src)
  const highlighted = highlightPreview(parts.content, keywords)
  currentSource.value = {
    fileName: parts.fileName,
    highlightedContent: highlighted
  }
  drawerVisible.value = true
}

const parseSource = (src) => {
  let fileName = ''
  let content = ''
  let score = '引用'
  let type = 'info'
  let shortContent = ''

  const splitIdx = src.indexOf('|')
  if (splitIdx !== -1) {
    fileName = src.substring(0, splitIdx).trim()
    content = src.substring(splitIdx + 1).trim()
  } else {
    fileName = src.replace('📄', '').trim()
    content = "暂无详细内容"
  }

  let header = fileName
  const scoreMatch = header.match(/\(匹配度: (\d+)%\)/)
  if (scoreMatch) {
    const val = parseInt(scoreMatch[1])
    score = val + '%'
    if (val >= 60) type = 'success'
    else if (val >= 50) type = 'warning'
    else type = 'info'
    fileName = header.replace(scoreMatch[0], '').trim()
  } else if (header.includes('关键词命中')) {
    score = '关键词'
    type = 'warning'
    fileName = header.replace('(关键词命中)', '').trim()
  } else if (header.includes('上下文扩展')) {
    score = '邻居'
    type = 'info'
    fileName = header.replace('(上下文扩展)', '').trim()
  }

  fileName = fileName.replace('📄', '').trim()

  shortContent = content
  if (shortContent.length > 25) {
    shortContent = shortContent.substring(0, 25) + '...'
  }

  return { fileName, score, type, content, shortContent }
}

const highlightPreview = (text, keywords) => {
  if (!text) return ''
  let res = text
  if (keywords && keywords.length > 0) {
    const sortedKws = [...keywords].sort((a, b) => b.length - a.length)
    sortedKws.forEach(kw => {
      if (kw && kw.trim().length > 0) {
        try {
          const reg = new RegExp(`(${kw})`, 'gi')
          res = res.replace(reg, `<span class="highlight-kw">$1</span>`)
        } catch (e) {}
      }
    })
  }
  return res
}

const copyToClipboard = async (text) => {
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success('内容已复制')
  } catch (e) {
    ElMessage.error('复制失败')
  }
}

const handleRegenerate = () => {
  let lastUserMsg = ''
  for (let i = messages.value.length - 1; i >= 0; i--) {
    if (messages.value[i].role === 'user') {
      lastUserMsg = messages.value[i].content
      break
    }
  }
  if (!lastUserMsg) return
  if (messages.value.length > 0 && messages.value[messages.value.length - 1].role === 'assistant') {
    messages.value.pop()
  }
  handleSend(lastUserMsg)
}

const submitFeedbackApi = async (chatId, type, reason) => {
  const response = await fetch(`${BASE_URL}/api/v1/feedback/submit`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${getToken()}` },
    body: JSON.stringify({ chat_id: chatId, type, reason })
  })
  if (!response.ok) throw new Error('反馈提交失败')
  return await response.json()
}

const handleLike = async (msg) => {
  if (!msg.id) return
  const newType = msg.feedbackType === 1 ? 0 : 1
  try {
    const oldType = msg.feedbackType
    msg.feedbackType = newType
    await submitFeedbackApi(msg.id, newType, "")
    if (newType === 1) ElMessage.success("感谢您的鼓励！")
    else ElMessage.info("已取消点赞")
  } catch (e) {
    msg.feedbackType = oldType
    ElMessage.error("操作失败")
  }
}

const handleDislike = async (msg) => {
  if (!msg.id) return
  if (msg.feedbackType === 2) {
    try {
      const oldType = msg.feedbackType
      msg.feedbackType = 0
      await submitFeedbackApi(msg.id, 0, "")
      ElMessage.info("已取消反馈")
    } catch (e) {
      msg.feedbackType = 2
      ElMessage.error("操作失败")
    }
    return
  }
  currentFeedbackMsg.value = msg
  feedbackReason.value = ""
  feedbackDialogVisible.value = true
}

const submitDislike = async () => {
  if (!feedbackReason.value.trim()) {
    ElMessage.warning("请填写反馈原因")
    return
  }
  if (!currentFeedbackMsg.value) return
  try {
    await submitFeedbackApi(currentFeedbackMsg.value.id, 2, feedbackReason.value)
    currentFeedbackMsg.value.feedbackType = 2
    feedbackDialogVisible.value = false
    ElMessage.success("反馈已提交")
  } catch (e) {
    ElMessage.error("提交失败")
  }
}

const getToken = () => localStorage.getItem('token') || ''

const handleSend = async (retryContent = null) => {
  const question = retryContent || input.value
  if (!question.trim() || loading.value) return

  if (!retryContent) {
    input.value = ''
    messages.value.push({role: 'user', content: question})
  }

  scrollToBottom()

  const aiMsgIndex = messages.value.push({
    role: 'assistant', content: '', thinking: null,
    id: null,
    feedbackType: 0
  }) - 1

  loading.value = true

  try {
    const response = await fetch(`${BASE_URL}/api/v1/chat`, {
      method: 'POST',
      headers: {'Content-Type': 'application/json', 'Authorization': `Bearer ${getToken()}`},
      body: JSON.stringify({question: question, conversation_id: currentConversationId.value})
    })

    if (!response.ok) throw new Error('网络请求失败')

    const reader = response.body.getReader()
    const decoder = new TextDecoder()
    let buffer = ''
    let isThinkingParsed = false

    while (true) {
      const {done, value} = await reader.read()
      if (done) break

      const chunk = decoder.decode(value, {stream: true})
      buffer += chunk

      if (buffer.includes("CONF_ID:")) {
        const match = buffer.match(/CONF_ID:(\d+)/)
        if (match) {
          const newId = parseInt(match[1])
          if (currentConversationId.value === 0 || currentConversationId.value !== newId) {
            currentConversationId.value = newId
            await fetchConversations()
          }
          buffer = buffer.replace(/CONF_ID:\d+/, '')
        }
      }

      if (!isThinkingParsed && buffer.includes("THINKING:")) {
        const startIdx = buffer.indexOf("THINKING:")
        const endIdx = buffer.indexOf("\n", startIdx)
        if (endIdx !== -1) {
          const jsonStr = buffer.substring(startIdx + 9, endIdx)
          try {
            messages.value[aiMsgIndex].thinking = JSON.parse(jsonStr)
            isThinkingParsed = true
            buffer = buffer.substring(0, startIdx) + buffer.substring(endIdx + 1)
          } catch (e) {
            buffer = buffer.substring(endIdx + 1)
            isThinkingParsed = true
          }
        }
      }

      if (isThinkingParsed || !buffer.trim().startsWith("THINKING:")) {
        if (buffer) {
          messages.value[aiMsgIndex].content += buffer
          buffer = ''
        }
      }
      scrollToBottom()
    }

  } catch (error) {
    messages.value[aiMsgIndex].content += "\n[网络或服务异常]"
  } finally {
    loading.value = false
    scrollToBottom()
    if (currentConversationId.value) {
      await refreshCurrentMessages()
    }
  }
}

const refreshCurrentMessages = async () => {
  try {
    const res = await getConversationMessages(currentConversationId.value)
    if (res && res.length > 0) {
      const lastItem = res[res.length - 1]
      if (messages.value.length > 0) {
        const lastLocal = messages.value[messages.value.length - 1]
        if (lastLocal.role === 'assistant') {
          lastLocal.id = lastItem.ID || lastItem.id
        }
      }
    }
  } catch (e) {}
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
  } catch (e) {
  }
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
.chat-container { display: flex; height: 100vh; background-color: #f4f4f7; transition: background-color 0.3s; }
.sidebar { width: 300px; background: #fff; border-right: 1px solid #e0e0e0; display: flex; flex-direction: column; transition: all 0.3s; }
.sidebar.collapsed { width: 60px; }
.sidebar-header { padding: 20px 15px; display: flex; gap: 10px; align-items: center; border-bottom: 1px solid #eee; }
.new-chat-btn { flex: 1; font-weight: bold; font-size: 15px; }
.sidebar.collapsed .new-chat-btn { display: none; }
.history-list { flex: 1; overflow-y: auto; padding: 15px 10px; }

.history-item { display: flex; align-items: center; justify-content: space-between; padding: 15px; cursor: pointer; border-radius: 8px; color: #666; font-size: 16px; margin-bottom: 8px; transition: all 0.2s; }
.history-item:hover { background: #f0f2f5; color: #333; }
.history-item.active { background: #e6f7ff; color: #409EFF; font-weight: bold; }
.history-content { display: flex; align-items: center; gap: 12px; overflow: hidden; }
.history-title { white-space: nowrap; overflow: hidden; text-overflow: ellipsis; max-width: 170px; }
.sidebar.collapsed .history-title { display: none; }
.history-actions { opacity: 0; pointer-events: none; padding: 2px; transition: opacity 0.2s; }
.history-item:hover .history-actions { opacity: 1; pointer-events: auto; }
.more-btn { font-size: 18px; color: #999; transform: rotate(90deg); }
.more-btn:hover { color: #409EFF; }
.sidebar-footer { padding: 20px; border-top: 1px solid #eee; text-align: center; }

.main-chat { flex: 1; display: flex; flex-direction: column; position: relative; background: #f4f4f7; transition: background 0.3s; }
.chat-header { height: 70px; background: #fff; border-bottom: 1px solid #eee; display: flex; align-items: center; justify-content: space-between; padding: 0 25px; transition: background 0.3s, border 0.3s; }
.chat-header h2 { font-size: 22px; color: #303133; margin: 0; }
.header-right { display: flex; align-items: center; }
.user-info-trigger { display: flex; align-items: center; gap: 12px; cursor: pointer; padding: 5px 12px; border-radius: 20px; transition: background 0.3s; }
.user-info-trigger:hover { background: rgba(0, 0, 0, 0.05); }
.username { font-size: 16px; font-weight: 500; color: #333; transition: color 0.3s; }

.announcement-banner { display: flex; align-items: center; background-color: #fdf6ec; color: #e6a23c; padding: 12px 20px; border-bottom: 1px solid #faecd8; }
.banner-icon { margin-right: 15px; font-size: 24px; display: flex; align-items: center; }
.marquee-container { flex: 1; overflow: hidden; white-space: nowrap; position: relative; }
.marquee-content { display: inline-block; animation: marquee 15s linear infinite; font-size: 16px; letter-spacing: 0.5px; }
.marquee-content:hover { animation-play-state: paused; }
@keyframes marquee { 0% { transform: translateX(100%); } 100% { transform: translateX(-100%); } }

.messages { flex: 1; padding: 25px 30px; overflow-y: auto; display: flex; flex-direction: column; gap: 25px; }
.welcome-box { text-align: center; margin-top: 10vh; color: #999; max-width: 800px; margin-left: auto; margin-right: auto; }
.welcome-box h3 { font-size: 24px; color: #606266; }
.welcome-box p { font-size: 16px; margin-top: 10px; }

/* 🚀 快捷指令卡片样式 */
.prompt-cards { display: grid; grid-template-columns: repeat(2, 1fr); gap: 15px; margin-top: 30px; text-align: left; }
.prompt-card { background: #fff; border: 1px solid #e4e7ed; border-radius: 10px; padding: 18px; cursor: pointer; transition: all 0.3s; }
.prompt-card:hover { border-color: #409EFF; box-shadow: 0 4px 12px rgba(64,158,255,0.1); transform: translateY(-2px); }
.prompt-card h4 { margin: 0 0 10px 0; color: #303133; font-size: 16px; display: flex; align-items: center; gap: 6px; }
.prompt-card p { margin: 0; color: #909399; font-size: 14px; line-height: 1.5; display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden; }

.message-row { display: flex; gap: 18px; max-width: 85%; }
.message-row.user { align-self: flex-end; flex-direction: row-reverse; }

.bubble-container { display: flex; flex-direction: column; gap: 8px; max-width: 100%; }
.message-row .bubble { background: #fff; padding: 14px 20px; border-radius: 14px; box-shadow: 0 2px 10px rgba(0, 0, 0, 0.05); line-height: 1.7; font-size: 16px; white-space: pre-wrap; transition: background 0.3s, color 0.3s; }
.message-row.user .bubble { background: #409EFF; color: #fff; border-radius: 14px 14px 0 14px; }
.message-row.assistant .bubble { border-radius: 0 14px 14px 14px; }

.input-area { background: #fff; padding: 25px; border-top: 1px solid #eee; transition: background 0.3s, border 0.3s; }
.input-box { max-width: 900px; margin: 0 auto; display: flex; gap: 15px; position: relative; }
:deep(.el-textarea__inner) { font-size: 16px !important; padding: 12px 15px; line-height: 1.6; }
.send-btn { position: absolute; right: 12px; bottom: 8px; transform: scale(1.1); }

.thinking-panel { background: #fdfdfd; border: 1px solid #e4e7ed; border-radius: 10px; overflow: hidden; margin-bottom: 8px; box-shadow: 0 1px 4px rgba(0, 0, 0, 0.03); }
:deep(.el-collapse) { border: none; }
:deep(.el-collapse-item__header) { background: #f9fafe; padding: 0 15px; height: 42px; line-height: 42px; font-size: 14px; color: #606266; border-bottom: 1px solid #ebeef5; }
:deep(.el-collapse-item__content) { padding: 15px; background: #fff; }
.thinking-title { font-weight: 600; color: #606266; font-size: 14px; }
.think-content { font-size: 14px; color: #555; display: flex; flex-direction: column; gap: 10px; }
.think-row { display: flex; align-items: flex-start; gap: 10px; }
.think-row .label { color: #909399; white-space: nowrap; min-width: 80px; text-align-last: justify; text-align: justify; margin-right: 5px; }
.think-row .value { flex: 1; line-height: 1.5; }
.think-row .value.strong { font-weight: 500; color: #303133; }
.think-row .value.code { font-family: Consolas, monospace; background: #f4f4f5; padding: 2px 6px; border-radius: 4px; color: #E6A23C; }
.tags { display: flex; flex-wrap: wrap; gap: 6px; }

.source-list { list-style: none; padding: 0; margin: 0; width: 100%; }
.source-list .source-item { cursor: pointer; transition: all 0.2s; background: #f0f9eb; padding: 4px 8px; border-radius: 6px; margin-bottom: 6px; font-size: 13px; display: flex; align-items: center; gap: 8px; max-width: 100%; }
.source-list .source-item:hover { filter: brightness(0.95); }
.mini-tag { flex-shrink: 0; height: 20px; line-height: 18px; padding: 0 6px; font-size: 12px; }
.source-text { flex: 1; color: #606266; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; display: flex; align-items: center; }
.fname { font-weight: bold; color: #333; margin-right: 4px; }
.sep { margin: 0 4px; color: #bbb; }
.preview { color: #888; }

.feedback-actions { display: flex; gap: 18px; margin-top: 8px; padding-left: 10px; opacity: 0; transition: opacity 0.2s ease-in-out; height: 26px; align-items: center; }
.bubble-container:hover .feedback-actions { opacity: 1; }
.action-icon { cursor: pointer; font-size: 18px; color: #606266; transition: all 0.2s; display: flex; align-items: center; }
.action-icon.disabled { cursor: not-allowed; opacity: 0.5; }
.action-icon:not(.disabled):hover { transform: scale(1.15); color: #409EFF; }
.like-btn:hover, .like-btn.active { color: #67C23A; }
.dislike-btn:hover, .dislike-btn.active { color: #F56C6C; }
.divider { width: 1px; height: 14px; background-color: #ddd; margin: 0 5px; }

.think-footer { margin-top: 8px; padding-top: 8px; border-top: 1px dashed #eee; color: #c0c4cc; font-size: 13px; text-align: right; }
.typing-cursor::after { content: '▋'; animation: blink 1s infinite; }
@keyframes blink { 50% { opacity: 0; } }

/* 🚀 黑夜模式适配增强 */
html.dark .chat-container { background-color: transparent; }
html.dark .main-chat { background-color: transparent; }
html.dark .sidebar { background: rgba(20, 20, 20, 0.6); backdrop-filter: blur(10px); border-right: 1px solid rgba(255, 255, 255, 0.1); }
html.dark .sidebar-header { border-bottom: 1px solid rgba(255, 255, 255, 0.1); color: #ccc; }
html.dark .sidebar-footer { border-top: 1px solid rgba(255, 255, 255, 0.1); }
html.dark .history-item { color: #a0a0a0; }
html.dark .history-item:hover { background: rgba(255, 255, 255, 0.1); color: #fff; }
html.dark .history-item.active { background: rgba(64, 158, 255, 0.2); color: #409EFF; }
html.dark .chat-header { background: rgba(20, 20, 20, 0.6); backdrop-filter: blur(10px); border-bottom: 1px solid rgba(255, 255, 255, 0.1); }
html.dark .chat-header h2 { color: #e0e0e0; }
html.dark .username { color: #ccc; }

html.dark .announcement-banner { background-color: rgba(230, 162, 60, 0.1); border-bottom: 1px solid rgba(230, 162, 60, 0.2); }

html.dark .input-area { background: rgba(20, 20, 20, 0.6); backdrop-filter: blur(10px); border-top: 1px solid rgba(255, 255, 255, 0.1); }
html.dark .message-row.assistant .bubble { background: rgba(45, 45, 45, 0.85); color: #e0e0e0; border: 1px solid rgba(255, 255, 255, 0.1); }
html.dark .input-box :deep(.el-textarea__inner) { background-color: rgba(45, 45, 45, 0.5) !important; color: #fff !important; border: 1px solid rgba(255, 255, 255, 0.2); }
html.dark .welcome-box h3, html.dark .welcome-box p { color: #bbb; }
html.dark .action-icon { color: #aaa; }

/* 黑夜模式的卡片适配 */
html.dark .prompt-card { background: rgba(30, 30, 30, 0.6); border-color: rgba(255,255,255,0.1); }
html.dark .prompt-card h4 { color: #E0E0E0; }
html.dark .prompt-card p { color: #888; }
html.dark .prompt-card:hover { border-color: #409EFF; }
</style>

<style>
.highlight-kw { background-color: #ffeb3b; color: #d32f2f; font-weight: bold; padding: 0 3px; border-radius: 3px; }
html.dark .highlight-kw { background-color: rgba(230, 162, 60, 0.3); color: #ff9800; }
html.dark .source-text { color: #e0e0e0 !important; }
html.dark .fname { color: #ffffff !important; }
html.dark .preview { color: #aaaaaa !important; }
html.dark .sep { color: #666666 !important; }

html.dark .thinking-panel { background: rgba(0, 0, 0, 0.4) !important; backdrop-filter: blur(10px); border: 1px solid rgba(255, 255, 255, 0.15) !important; box-shadow: none !important; color: #E0E0E0; }
html.dark .thinking-panel .el-collapse { border: none !important; background: transparent !important; --el-collapse-header-bg-color: transparent !important; --el-collapse-content-bg-color: transparent !important; }
html.dark .thinking-panel .el-collapse-item__header { background: rgba(255, 255, 255, 0.05) !important; color: #E0E0E0 !important; border-bottom: 1px solid rgba(255, 255, 255, 0.1) !important; padding-left: 15px !important; font-size: 14px !important; }
html.dark .thinking-panel .el-collapse-item__wrap { background: transparent !important; border: none !important; }
html.dark .thinking-panel .el-collapse-item__content { background: transparent !important; color: #ccc !important; padding: 15px !important; }
html.dark .thinking-panel .think-row .label { color: #888 !important; }
html.dark .thinking-panel .think-row .value { color: #ddd !important; }
html.dark .thinking-panel .think-row .value.strong { color: #fff !important; text-shadow: 0 0 10px rgba(64, 158, 255, 0.6); }
html.dark .thinking-panel .think-row .value.code { background: rgba(255, 255, 255, 0.1) !important; color: #E6A23C !important; border: 1px solid rgba(255, 255, 255, 0.1) !important; }
html.dark .thinking-panel .tags .el-tag { background-color: rgba(255, 255, 255, 0.1) !important; border-color: rgba(255, 255, 255, 0.2) !important; color: #eee !important; }
html.dark .thinking-panel .source-list li { background: rgba(103, 194, 58, 0.15) !important; color: #95d475 !important; border: 1px solid rgba(103, 194, 58, 0.3) !important; }
html.dark .thinking-panel .think-footer { border-top: 1px dashed rgba(255, 255, 255, 0.15) !important; color: #666 !important; }

/* 🚀 右侧抽屉加大字体 */
html.dark .el-drawer { background-color: rgba(30, 30, 30, 0.95) !important; color: #e0e0e0; }
html.dark .el-drawer__header { color: #e0e0e0; border-bottom: 1px solid rgba(255, 255, 255, 0.1); margin-bottom: 0; font-size: 18px !important; }
.raw-content-box { background: #f4f6f8; padding: 20px; border-radius: 10px; border: 1px solid #eee; line-height: 1.8; font-size: 16px; margin-top: 15px; }
html.dark .raw-content-box { background: rgba(0, 0, 0, 0.3); border: 1px solid rgba(255, 255, 255, 0.1); color: #ddd; }
</style>