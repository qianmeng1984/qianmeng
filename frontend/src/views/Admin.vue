<template>
  <div class="admin-container">
    <div class="content-wrapper">

      <div class="page-header">
        <div class="header-left">
          <h1>🛡️ 系统管理控制台</h1>
          <p class="subtitle">RAG 知识库与底层管控中心</p>
        </div>
        <el-button type="primary" plain size="large" @click="router.push('/')">
          <el-icon style="margin-right: 5px"><Back /></el-icon> 返回对话系统
        </el-button>
      </div>

      <el-tabs v-model="activeTab" class="admin-tabs menu-layout" type="border-card" tab-position="left">

        <el-tab-pane name="dashboard">
          <template #label><div class="menu-item"><el-icon><DataLine /></el-icon> 数据仪表盘</div></template>

          <div class="stat-cards">
            <el-card shadow="hover" class="stat-item">
              <div class="stat-icon bg-blue"><el-icon><Document /></el-icon></div>
              <div class="stat-info">
                <div class="stat-value">{{ stats.total_files || 0 }}</div>
                <div class="stat-label">收录文档数</div>
              </div>
            </el-card>
            <el-card shadow="hover" class="stat-item">
              <div class="stat-icon bg-green"><el-icon><Cpu /></el-icon></div>
              <div class="stat-info">
                <div class="stat-value">{{ stats.total_chunks || 0 }}</div>
                <div class="stat-label">向量切片总数</div>
              </div>
            </el-card>
            <el-card shadow="hover" class="stat-item">
              <div class="stat-icon bg-purple"><el-icon><User /></el-icon></div>
              <div class="stat-info">
                <div class="stat-value">{{ userList.length }}</div>
                <div class="stat-label">注册用户数</div>
              </div>
            </el-card>
          </div>

          <div class="chart-row">
            <el-card class="chart-card" shadow="never">
              <template #header><div class="card-title">📚 知识库内容分布</div></template>
              <div ref="pieChartRef" class="chart-box"></div>
            </el-card>
            <el-card class="chart-card" shadow="never">
              <template #header><div class="card-title">🔥 学生提问热词 Top 10</div></template>
              <div ref="barChartRef" class="chart-box"></div>
            </el-card>
          </div>
        </el-tab-pane>

        <el-tab-pane name="users">
          <template #label><div class="menu-item"><el-icon><Avatar /></el-icon> 用户权限管理</div></template>
          <el-card class="table-card" shadow="never">
            <template #header><div class="card-title">👥 系统账户一览</div></template>
            <el-table :data="userList" stripe style="width: 100%" v-loading="loading">
              <el-table-column prop="ID" label="ID" width="100" align="center" />
              <el-table-column label="头像" width="100" align="center">
                <template #default="scope">
                  <el-avatar :size="40" :src="getFullUrl(scope.row.Avatar)" />
                </template>
              </el-table-column>
              <el-table-column prop="Username" label="账号" />
              <el-table-column prop="Nickname" label="昵称" />
              <el-table-column label="角色" width="150">
                <template #default="scope">
                  <el-tag :type="scope.row.Role === 1 ? 'danger' : 'info'" effect="dark" size="large">
                    {{ scope.row.Role === 1 ? '超级管理员' : '普通用户' }}
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column label="操作" align="right">
                <template #default="scope">
                  <el-button size="default" :icon="Edit" @click="openEdit(scope.row)">编辑</el-button>
                  <el-popconfirm title="确定注销该用户？" @confirm="handleDelete(scope.row.ID)">
                    <template #reference>
                      <el-button size="default" type="danger" :icon="Delete" :disabled="scope.row.Role === 1">注销</el-button>
                    </template>
                  </el-popconfirm>
                </template>
              </el-table-column>
            </el-table>
          </el-card>
        </el-tab-pane>

        <el-tab-pane name="sensitive">
          <template #label><div class="menu-item"><el-icon><Lock /></el-icon> 敏感词与合规</div></template>
          <el-card class="table-card" shadow="never">
            <template #header>
              <div class="card-header-flex">
                <div class="card-title">🚫 敏感词拦截策略库</div>
                <el-button type="primary" size="large" :icon="Plus" @click="openSensitiveDialog()">新增拦截词</el-button>
              </div>
            </template>
            <el-alert title="安全提示" type="warning" show-icon style="margin-bottom: 25px; padding: 15px;" :closable="false">
              阻断级别 (Level 2) 将直接拒绝回答；警告级别 (Level 1) 将在发送给大模型前自动将该词替换为 ***。
            </el-alert>
            <el-table :data="sensitiveList" stripe style="width: 100%" v-loading="loadingSensitive">
              <el-table-column prop="ID" label="规则ID" width="100" align="center" />
              <el-table-column prop="word" label="触发词 / 违规词汇">
                <template #default="{ row }">
                  <span style="font-family: monospace; font-weight: bold; font-size: 16px;">{{ row.word }}</span>
                </template>
              </el-table-column>
              <el-table-column label="处理策略" width="220">
                <template #default="{ row }">
                  <el-tag :type="row.level === 2 ? 'danger' : 'warning'" effect="dark" size="large">
                    {{ row.level === 2 ? '直接阻断请求' : '替换为 *** (警告)' }}
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column label="操作" align="right" width="220">
                <template #default="scope">
                  <el-button size="default" :icon="Edit" @click="openSensitiveDialog(scope.row)">编辑</el-button>
                  <el-popconfirm title="确定移除该规则？" @confirm="deleteSensitive(scope.row.ID)">
                    <template #reference>
                      <el-button size="default" type="danger" :icon="Delete" plain>移除</el-button>
                    </template>
                  </el-popconfirm>
                </template>
              </el-table-column>
            </el-table>
          </el-card>
        </el-tab-pane>

        <el-tab-pane name="announcement">
          <template #label><div class="menu-item"><el-icon><Bell /></el-icon> 系统公告发布</div></template>
          <el-card class="table-card" shadow="never">
            <template #header>
              <div class="card-header-flex">
                <div class="card-title">📢 全局滚动公告管理</div>
                <el-button type="primary" size="large" :icon="Plus" @click="openAnnounceDialog()">发布新公告</el-button>
              </div>
            </template>
            <el-table :data="announceList" stripe style="width: 100%" v-loading="loadingAnnounce">
              <el-table-column prop="ID" label="ID" width="80" align="center" />
              <el-table-column prop="title" label="公告标题" width="250" show-overflow-tooltip>
                <template #default="{ row }"><strong>{{ row.title }}</strong></template>
              </el-table-column>
              <el-table-column prop="content" label="公告内容摘要" show-overflow-tooltip />
              <el-table-column label="发布状态" width="150" align="center">
                <template #default="{ row }">
                  <el-switch v-model="row.status" :active-value="1" :inactive-value="0"
                             inline-prompt active-text="已发布" inactive-text="草稿"
                             @change="toggleAnnounceStatus(row)" />
                </template>
              </el-table-column>
              <el-table-column label="操作" align="right" width="200">
                <template #default="scope">
                  <el-button size="default" :icon="Edit" @click="openAnnounceDialog(scope.row)">编辑</el-button>
                  <el-popconfirm title="确定删除该公告？" @confirm="deleteAnnounce(scope.row.ID)">
                    <template #reference>
                      <el-button size="default" type="danger" :icon="Delete" plain>删除</el-button>
                    </template>
                  </el-popconfirm>
                </template>
              </el-table-column>
            </el-table>
          </el-card>
        </el-tab-pane>

        <el-tab-pane name="prompts">
          <template #label><div class="menu-item"><el-icon><ChatLineSquare /></el-icon> 快捷指令库</div></template>
          <el-card class="table-card" shadow="never">
            <template #header>
              <div class="card-header-flex">
                <div class="card-title">💡 首页冷启动指令推荐配置</div>
                <el-button type="primary" size="large" :icon="Plus" @click="openPromptDialog()">新增指令</el-button>
              </div>
            </template>
            <el-alert title="运营提示" type="info" show-icon style="margin-bottom: 25px; padding: 15px;" :closable="false">
              配置的快捷指令将显示在学生打开对话的欢迎页中，帮助解决“冷启动难题”。排序权重越大越靠前。
            </el-alert>
            <el-table :data="promptList" stripe style="width: 100%" v-loading="loadingPrompt">
              <el-table-column prop="ID" label="ID" width="80" align="center" />
              <el-table-column prop="title" label="指令标题" width="200">
                <template #default="{ row }"><strong>{{ row.title }}</strong></template>
              </el-table-column>
              <el-table-column prop="content" label="完整提问内容" show-overflow-tooltip />
              <el-table-column prop="sort_weight" label="权重" width="100" align="center">
                <template #default="{ row }"><el-tag type="info">{{ row.sort_weight }}</el-tag></template>
              </el-table-column>
              <el-table-column label="状态" width="120" align="center">
                <template #default="{ row }">
                  <el-switch v-model="row.is_active" :active-value="1" :inactive-value="0" @change="togglePromptStatus(row)" />
                </template>
              </el-table-column>
              <el-table-column label="操作" align="right" width="200">
                <template #default="scope">
                  <el-button size="default" :icon="Edit" @click="openPromptDialog(scope.row)">编辑</el-button>
                  <el-popconfirm title="确定删除该指令？" @confirm="deletePrompt(scope.row.ID)">
                    <template #reference><el-button size="default" type="danger" :icon="Delete" plain>删除</el-button></template>
                  </el-popconfirm>
                </template>
              </el-table-column>
            </el-table>
          </el-card>
        </el-tab-pane>

        <el-tab-pane name="blind_spot">
          <template #label><div class="menu-item"><el-icon><Aim /></el-icon> 知识盲区监控</div></template>
          <el-card class="table-card" shadow="never">
            <template #header>
              <div class="card-header-flex">
                <div class="card-title">🎯 知识库盲区自动捕获与闭环</div>
              </div>
            </template>
            <el-alert title="数据飞轮 (Data Flywheel) 提示" type="error" show-icon style="margin-bottom: 25px; padding: 15px;" :closable="false">
              系统会自动拦截并记录所有检索命中切片为 0 的提问（Bad Case）。请根据学生高频提问的盲区，有针对性地上传文档，补全知识库。
            </el-alert>

            <el-table :data="blindSpotList" stripe style="width: 100%" v-loading="loadingBlindSpot">
              <el-table-column prop="ID" label="工单ID" width="100" align="center" />
              <el-table-column prop="question" label="未命中检索的原始提问 (Bad Case)">
                <template #default="{ row }">
                  <strong style="color: #F56C6C;">"{{ row.question }}"</strong>
                </template>
              </el-table-column>
              <el-table-column label="处理状态" width="180" align="center">
                <template #default="{ row }">
                  <el-tag :type="row.status === 1 ? 'success' : 'danger'" effect="dark" size="large">
                    {{ row.status === 1 ? '已补充文档' : '待补充知识' }}
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column label="操作" align="right" width="300">
                <template #default="scope">
                  <el-button size="default" :icon="View" @click="viewBlindSpotContext(scope.row)">回放上下文</el-button>
                  <el-button size="default" type="success" plain :disabled="scope.row.status === 1" @click="resolveBlindSpot(scope.row.ID)">标记解决</el-button>
                  <el-popconfirm title="确定忽略此盲区工单？" @confirm="deleteBlindSpot(scope.row.ID)">
                    <template #reference><el-button size="default" type="danger" :icon="Delete" plain>删除</el-button></template>
                  </el-popconfirm>
                </template>
              </el-table-column>
            </el-table>
          </el-card>
        </el-tab-pane>

        <el-tab-pane name="feedback">
          <template #label><div class="menu-item"><el-icon><Service /></el-icon> 反馈处理中心</div></template>
          <div class="feedback-table-wrapper">
            <el-table :data="feedbackList" style="width: 100%" v-loading="loadingFeedback" size="large">
              <el-table-column prop="user_id" label="ID" width="80" />
              <el-table-column label="提交用户" width="200">
                <template #default="{ row }">
                  <div class="user-cell">
                    <el-avatar :size="32" :src="getFullUrl((row.User || row.user)?.Avatar || (row.User || row.user)?.avatar)" />
                    <span>{{ (row.User || row.user)?.Nickname || (row.User || row.user)?.nickname || (row.User || row.user)?.Username || '未知' }}</span>
                  </div>
                </template>
              </el-table-column>
              <el-table-column label="反馈类型" width="120">
                <template #default="{ row }">
                  <el-tag type="danger" v-if="row.type === 2 || row.Type === 2" effect="dark" size="large">
                    <el-icon><CircleCloseFilled /></el-icon> 点踩
                  </el-tag>
                  <el-tag type="success" v-else effect="dark" size="large">
                    <el-icon><SuccessFilled /></el-icon> 点赞
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column label="关联问题摘要" show-overflow-tooltip>
                <template #default="{ row }">
                  {{ row.ChatHistory?.question || row.ChatHistory?.Question || row.chat_history?.question || '（原问题已删除）' }}
                </template>
              </el-table-column>
              <el-table-column prop="reason" label="反馈原因" show-overflow-tooltip />
              <el-table-column label="状态" width="120">
                <template #default="{ row }">
                  <el-tag v-if="row.status === 1 || row.Status === 1" type="success" size="large">已处理</el-tag>
                  <el-tag v-else type="warning" size="large">待处理</el-tag>
                </template>
              </el-table-column>
              <el-table-column label="操作" width="140" fixed="right">
                <template #default="{ row }">
                  <el-button type="primary" size="default" @click="handleFeedback(row)">处理 / 查看</el-button>
                </template>
              </el-table-column>
            </el-table>
          </div>
        </el-tab-pane>

      </el-tabs>
    </div>

    <el-dialog v-model="dialogVisible" title="修改用户信息" width="500px" align-center class="large-dialog">
      <el-form :model="editForm" label-width="90px" size="large">
        <el-form-item label="账号"><el-input v-model="editForm.username" disabled /></el-form-item>
        <el-form-item label="昵称"><el-input v-model="editForm.nickname" /></el-form-item>
        <el-form-item label="重置密码"><el-input v-model="editForm.password" placeholder="留空不改" show-password /></el-form-item>
      </el-form>
      <template #footer><div class="dialog-footer"><el-button size="large" @click="dialogVisible = false">取消</el-button><el-button size="large" type="primary" @click="handleUpdate">保存修改</el-button></div></template>
    </el-dialog>

    <el-dialog v-model="sensitiveDialogVisible" :title="sensitiveForm.id ? '编辑拦截词' : '新增拦截词'" width="550px" align-center class="large-dialog">
      <el-form :model="sensitiveForm" label-width="100px" size="large">
        <el-form-item label="触发词"><el-input v-model="sensitiveForm.word" placeholder="请输入违规或需过滤的词汇" /></el-form-item>
        <el-form-item label="管控策略">
          <el-radio-group v-model="sensitiveForm.level">
            <el-radio :label="2" border>🚫 阻断请求</el-radio>
            <el-radio :label="1" border>⚠️ 替换为***</el-radio>
          </el-radio-group>
        </el-form-item>
      </el-form>
      <template #footer><div class="dialog-footer"><el-button size="large" @click="sensitiveDialogVisible = false">取消</el-button><el-button size="large" type="primary" @click="saveSensitive">确认保存</el-button></div></template>
    </el-dialog>

    <el-dialog v-model="announceDialogVisible" :title="announceForm.id ? '编辑公告' : '发布新公告'" width="600px" align-center class="large-dialog">
      <el-form :model="announceForm" label-width="90px" size="large">
        <el-form-item label="公告标题"><el-input v-model="announceForm.title" placeholder="如：国庆放假通知 / 知识库更新说明" /></el-form-item>
        <el-form-item label="公告内容">
          <el-input v-model="announceForm.content" type="textarea" :rows="5" placeholder="请输入详细公告内容..." />
        </el-form-item>
        <el-form-item label="立即发布">
          <el-switch v-model="announceForm.status" :active-value="1" :inactive-value="0" />
          <span style="margin-left: 10px; font-size: 13px; color: #999;">关闭则存为草稿，前台不可见</span>
        </el-form-item>
      </el-form>
      <template #footer><div class="dialog-footer"><el-button size="large" @click="announceDialogVisible = false">取消</el-button><el-button size="large" type="primary" @click="saveAnnounce">确认保存</el-button></div></template>
    </el-dialog>

    <el-dialog v-model="promptDialogVisible" :title="promptForm.id ? '编辑快捷指令' : '新增快捷指令'" width="600px" align-center class="large-dialog">
      <el-form :model="promptForm" label-width="100px" size="large">
        <el-form-item label="简短标题"><el-input v-model="promptForm.title" placeholder="如：查校训" /></el-form-item>
        <el-form-item label="完整提问">
          <el-input v-model="promptForm.content" type="textarea" :rows="3" placeholder="如：请告诉我河南科技学院的校训是什么？" />
        </el-form-item>
        <el-form-item label="排序权重"><el-input-number v-model="promptForm.sort_weight" :min="0" :max="999" /></el-form-item>
        <el-form-item label="启用状态"><el-switch v-model="promptForm.is_active" :active-value="1" :inactive-value="0" /></el-form-item>
      </el-form>
      <template #footer><div class="dialog-footer"><el-button size="large" @click="promptDialogVisible = false">取消</el-button><el-button size="large" type="primary" @click="savePrompt">确认保存</el-button></div></template>
    </el-dialog>

    <el-dialog v-model="blindSpotContextVisible" title="🧐 盲区工单：上下文案件重演" width="800px" class="large-dialog">
      <div class="chat-context-panel" style="height: 500px; border: none;">
        <div class="chat-window" v-loading="loadingContext">
          <el-empty v-if="contextMessages.length === 0" description="无法追溯上下文记录"></el-empty>
          <div v-for="(msg, idx) in contextMessages" :key="idx" class="chat-bubble-row">
            <div class="bubble user"><div class="role">User</div><div class="text">{{ msg.question || msg.Question }}</div></div>
            <div class="bubble ai"><div class="role">AI (浅梦)</div><div class="text">{{ msg.answer || msg.Answer }}</div></div>
          </div>
        </div>
      </div>
      <template #footer>
        <span class="dialog-footer">
          <el-button size="large" @click="blindSpotContextVisible = false">关闭预览</el-button>
        </span>
      </template>
    </el-dialog>

    <el-dialog v-model="feedbackDialogVisible" title="反馈工单处理" width="1200px" class="feedback-dialog" top="5vh">
      <div class="feedback-layout">
        <div class="chat-context-panel">
          <div class="panel-header">📝 现场回放 (上下文)</div>
          <div class="chat-window" v-loading="loadingContext">
            <el-empty v-if="contextMessages.length === 0" description="无法加载上下文"></el-empty>
            <div v-for="(msg, idx) in contextMessages" :key="idx" class="chat-bubble-row" :class="{ 'is-target': isTargetMessage(msg) }">
              <div class="bubble user"><div class="role">User</div><div class="text">{{ msg.question || msg.Question }}</div></div>
              <div class="bubble ai"><div class="role">AI (浅梦)</div><div class="text">{{ msg.answer || msg.Answer }}</div></div>
              <div class="target-mark" v-if="isTargetMessage(msg)">👈 用户对此条表示不满</div>
            </div>
          </div>
        </div>
        <div class="feedback-action-panel">
          <div class="panel-header">🔧 处理意见</div>
          <div class="info-block"><label>提交用户：</label><div class="val">{{ (currentFeedback?.User || currentFeedback?.user)?.Nickname || (currentFeedback?.User || currentFeedback?.user)?.Username || '未知' }}</div></div>
          <div class="info-block"><label>反馈原因：</label><div class="val reason">{{ currentFeedback?.reason || currentFeedback?.Reason }}</div></div>
          <div class="info-block"><label>管理员回复：</label><el-input v-model="replyContent" type="textarea" :rows="8" placeholder="请输入回复内容，用户将在个人中心看到..." /></div>
          <div class="action-footer"><el-button size="large" @click="feedbackDialogVisible = false">取消</el-button><el-button size="large" type="primary" @click="submitReply">提交回复并标记为解决</el-button></div>
        </div>
      </div>
    </el-dialog>

  </div>
</template>

<script setup>
import { ref, onMounted, nextTick, watch } from 'vue'
import { useRouter } from 'vue-router'
import request from '@/utils/request'
import { ElMessage } from 'element-plus'
import {
  Back, Document, Cpu, User, Edit, Delete, CircleCloseFilled, SuccessFilled,
  DataLine, Avatar, Lock, Service, Plus, Bell, ChatLineSquare, Aim, View
} from '@element-plus/icons-vue'
import * as echarts from 'echarts'

const router = useRouter()
const activeTab = ref('dashboard')

// Dashboard
const userList = ref([])
const stats = ref({ total_files: 0, total_chunks: 0, type_dist: [], hot_words: [] })
const loading = ref(false)
const dialogVisible = ref(false)
const editForm = ref({ id: 0, username: '', nickname: '', password: '' })
const pieChartRef = ref(null)
const barChartRef = ref(null)

// Shared Context Logic
const loadingContext = ref(false)
const contextMessages = ref([])

// Feedback
const loadingFeedback = ref(false)
const feedbackList = ref([])
const feedbackDialogVisible = ref(false)
const currentFeedback = ref(null)
const replyContent = ref('')

// Sensitive
const sensitiveList = ref([])
const loadingSensitive = ref(false)
const sensitiveDialogVisible = ref(false)
const sensitiveForm = ref({ id: null, word: '', level: 2 })

// Announcement
const announceList = ref([])
const loadingAnnounce = ref(false)
const announceDialogVisible = ref(false)
const announceForm = ref({ id: null, title: '', content: '', status: 1 })

// Prompts
const promptList = ref([])
const loadingPrompt = ref(false)
const promptDialogVisible = ref(false)
const promptForm = ref({ id: null, title: '', content: '', is_active: 1, sort_weight: 0 })

// 🚀 Blind Spot (盲区监控)
const blindSpotList = ref([])
const loadingBlindSpot = ref(false)
const blindSpotContextVisible = ref(false)

const BASE_URL = 'http://localhost:8080'
const getFullUrl = (path) => path ? (path.startsWith('http') ? path : BASE_URL + path) : ''

const initData = async () => {
  loading.value = true
  await Promise.all([fetchUsers(), fetchStats()])
  loading.value = false
  initCharts()
}

watch(activeTab, (newVal) => {
  if (newVal === 'feedback' && feedbackList.value.length === 0) fetchFeedbacks()
  else if (newVal === 'sensitive' && sensitiveList.value.length === 0) fetchSensitiveWords()
  else if (newVal === 'announcement' && announceList.value.length === 0) fetchAnnouncements()
  else if (newVal === 'prompts' && promptList.value.length === 0) fetchPrompts()
  else if (newVal === 'blind_spot' && blindSpotList.value.length === 0) fetchBlindSpots()
  else if (newVal === 'dashboard') setTimeout(initCharts, 200)
})

const fetchUsers = async () => { try { userList.value = await request.get('/admin/users') || [] } catch (e) { router.push('/') } }
const fetchStats = async () => { try { stats.value = await request.get('/admin/stats') || {} } catch (e) {} }

// ============== 🚀 盲区监控管理 ==============
const fetchBlindSpots = async () => {
  loadingBlindSpot.value = true
  try { blindSpotList.value = await request.get('/admin/blindspot/list') || [] } catch(e) {} finally { loadingBlindSpot.value = false }
}
const viewBlindSpotContext = async (row) => {
  blindSpotContextVisible.value = true
  contextMessages.value = []
  if (row.conversation_id) {
    loadingContext.value = true
    try { contextMessages.value = await request.get(`/admin/feedback/context?conversation_id=${row.conversation_id}`) || [] }
    catch(e) { ElMessage.error('无法加载上下文') } finally { loadingContext.value = false }
  } else {
    ElMessage.warning('记录中缺乏会话ID')
  }
}
const resolveBlindSpot = async (id) => {
  try { await request.post('/admin/blindspot/resolve', { id }); ElMessage.success('已标记为解决'); fetchBlindSpots() } catch(e){ ElMessage.error('操作失败') }
}
const deleteBlindSpot = async (id) => {
  try { await request.post('/admin/blindspot/delete', { id }); ElMessage.success('已删除工单'); fetchBlindSpots() } catch(e){}
}

// ============== 快捷指令管理 ==============
const fetchPrompts = async () => { loadingPrompt.value = true; try { promptList.value = await request.get('/admin/prompt/list') || [] } catch(e) {} finally { loadingPrompt.value = false } }
const openPromptDialog = (row = null) => {
  if (row) promptForm.value = { id: row.ID, title: row.title, content: row.content, is_active: row.is_active, sort_weight: row.sort_weight }
  else promptForm.value = { id: null, title: '', content: '', is_active: 1, sort_weight: 0 }
  promptDialogVisible.value = true
}
const savePrompt = async () => {
  if (!promptForm.value.title || !promptForm.value.content) return ElMessage.warning('标题和内容不能为空')
  try {
    if (promptForm.value.id) await request.post('/admin/prompt/update', promptForm.value)
    else await request.post('/admin/prompt/add', promptForm.value)
    ElMessage.success('保存成功'); promptDialogVisible.value = false; fetchPrompts()
  } catch(e) { ElMessage.error('保存失败') }
}
const deletePrompt = async (id) => { try { await request.post('/admin/prompt/delete', { id }); ElMessage.success('已删除'); fetchPrompts() } catch(e){} }
const togglePromptStatus = async (row) => { try { await request.post('/admin/prompt/update', row); ElMessage.success('状态已更新') } catch(e) { row.is_active = row.is_active === 1 ? 0 : 1; ElMessage.error('更新失败') } }

// ============== 公告管理 ==============
const fetchAnnouncements = async () => { loadingAnnounce.value = true; try { announceList.value = await request.get('/admin/announcement/list') || [] } catch(e) {} finally { loadingAnnounce.value = false } }
const openAnnounceDialog = (row = null) => {
  if (row) announceForm.value = { id: row.ID, title: row.title, content: row.content, status: row.status }
  else announceForm.value = { id: null, title: '', content: '', status: 1 }
  announceDialogVisible.value = true
}
const saveAnnounce = async () => {
  if (!announceForm.value.title.trim() || !announceForm.value.content.trim()) return ElMessage.warning('标题和内容均不能为空')
  try {
    if (announceForm.value.id) { await request.post('/admin/announcement/update', announceForm.value) }
    else { await request.post('/admin/announcement/add', announceForm.value) }
    ElMessage.success('保存成功'); announceDialogVisible.value = false; fetchAnnouncements()
  } catch (e) { ElMessage.error('保存失败') }
}
const deleteAnnounce = async (id) => { try { await request.post('/admin/announcement/delete', { id }); ElMessage.success('已删除'); fetchAnnouncements() } catch(e){} }
const toggleAnnounceStatus = async (row) => { try { await request.post('/admin/announcement/update', { id: row.ID, title: row.title, content: row.content, status: row.status }); ElMessage.success('状态已更新') } catch(e) { row.status = row.status === 1 ? 0 : 1; ElMessage.error('更新失败') } }

// ============== 敏感词 ==============
const fetchSensitiveWords = async () => { loadingSensitive.value = true; try { sensitiveList.value = await request.get('/admin/sensitive/list') || [] } catch(e) {} finally { loadingSensitive.value = false } }
const openSensitiveDialog = (row = null) => {
  if (row) sensitiveForm.value = { id: row.ID, word: row.word, level: row.level }
  else sensitiveForm.value = { id: null, word: '', level: 2 }
  sensitiveDialogVisible.value = true
}
const saveSensitive = async () => {
  if (!sensitiveForm.value.word.trim()) return ElMessage.warning('规则词汇不能为空')
  try {
    if (sensitiveForm.value.id) await request.post('/admin/sensitive/update', sensitiveForm.value)
    else await request.post('/admin/sensitive/add', sensitiveForm.value)
    ElMessage.success('保存成功'); sensitiveDialogVisible.value = false; fetchSensitiveWords()
  } catch (e) { ElMessage.error('保存失败') }
}
const deleteSensitive = async (id) => { try { await request.post('/admin/sensitive/delete', { id }); ElMessage.success('已移除'); fetchSensitiveWords() } catch(e) {} }

// ============== 反馈 ==============
const fetchFeedbacks = async () => { loadingFeedback.value = true; try { feedbackList.value = await request.get('/admin/feedback/list') || [] } catch(e) {} finally { loadingFeedback.value = false } }
const handleFeedback = async (row) => {
  currentFeedback.value = row
  replyContent.value = row.admin_reply || row.AdminReply || ''
  feedbackDialogVisible.value = true
  const history = row.ChatHistory || row.chat_history
  const convId = history?.conversation_id || history?.ConversationID
  if (convId) {
    loadingContext.value = true; try { contextMessages.value = await request.get(`/admin/feedback/context?conversation_id=${convId}`) || [] } catch(e) { ElMessage.error('无法加载上下文') } finally { loadingContext.value = false }
  } else { contextMessages.value = []; ElMessage.warning('会话ID丢失') }
}
const isTargetMessage = (msg) => { if (!currentFeedback.value) return false; const targetHistory = currentFeedback.value.ChatHistory || currentFeedback.value.chat_history; if (!targetHistory) return false; return (targetHistory.ID || targetHistory.id) === (msg.ID || msg.id) }
const submitReply = async () => {
  if (!replyContent.value.trim()) return ElMessage.warning('请输入回复内容')
  try { await request.post('/admin/feedback/reply', { id: currentFeedback.value.ID || currentFeedback.value.id, reply: replyContent.value }); ElMessage.success('回复成功'); feedbackDialogVisible.value = false; fetchFeedbacks() } catch(e) { ElMessage.error('操作失败') }
}

// ============== 图表 ==============
const initCharts = () => {
  nextTick(() => {
    const isDark = document.documentElement.classList.contains('dark')
    const textColor = isDark ? '#E0E0E0' : '#333'
    if (pieChartRef.value) {
      if (echarts.getInstanceByDom(pieChartRef.value)) echarts.getInstanceByDom(pieChartRef.value).dispose();
      const pieChart = echarts.init(pieChartRef.value)
      pieChart.setOption({ tooltip: { trigger: 'item' }, legend: { bottom: '0%', left: 'center', textStyle: { color: textColor } }, series: [{ name: '知识来源', type: 'pie', radius: ['40%', '70%'], avoidLabelOverlap: false, itemStyle: { borderRadius: 10, borderColor: isDark ? '#1e1e1e' : '#fff', borderWidth: 2 }, label: { show: false, position: 'center' }, emphasis: { label: { show: true, fontSize: 20, fontWeight: 'bold', color: textColor } }, data: stats.value.type_dist || [{ value: 0, name: '暂无数据' }] }] })
      window.addEventListener('resize', () => pieChart.resize())
    }
    if (barChartRef.value) {
      if (echarts.getInstanceByDom(barChartRef.value)) echarts.getInstanceByDom(barChartRef.value).dispose();
      const barChart = echarts.init(barChartRef.value)
      const topWords = (stats.value.hot_words || []).slice(0, 10).reverse()
      barChart.setOption({ tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' } }, grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true }, xAxis: { type: 'value', splitLine: { show: false }, axisLabel: { color: textColor } }, yAxis: { type: 'category', data: topWords.map(i => i.name), axisTick: { show: false }, axisLine: { show: false }, axisLabel: { color: textColor } }, series: [{ name: '提问频次', type: 'bar', data: topWords.map(i => i.value), barWidth: '60%', itemStyle: { color: new echarts.graphic.LinearGradient(0, 0, 1, 0, [{ offset: 0, color: '#83bff6' }, { offset: 0.5, color: '#188df0' }, { offset: 1, color: '#188df0' }]), borderRadius: [0, 4, 4, 0] }, label: { show: true, position: 'right', color: textColor } }] })
      window.addEventListener('resize', () => barChart.resize())
    }
  })
}

const handleDelete = async (id) => { try { await request.post('/admin/user/delete', { id }); ElMessage.success('已注销'); fetchUsers() } catch(e){} }
const openEdit = (row) => { editForm.value = { id: row.ID, username: row.Username, nickname: row.Nickname, password: '' }; dialogVisible.value = true }
const handleUpdate = async () => { try { await request.post('/admin/user/update', editForm.value); ElMessage.success('修改成功'); dialogVisible.value = false; fetchUsers() } catch(e) { ElMessage.error('失败') } }

onMounted(() => { initData() })
</script>

<style scoped>
/* =========== 核心修正：左侧菜单强行左对齐 =========== */
:deep(.el-tabs__item) {
  height: 60px;
  line-height: 60px;
  font-size: 16px;
  text-align: left;
  padding-left: 25px !important;
  color: #606266;
  border: none !important;
  transition: all 0.3s;
  display: flex !important;              /* 强制开启 Flex 布局 */
  justify-content: flex-start !important; /* 强制左对齐 */
}
/* ================================================= */

.admin-container { min-height: 100vh; background-color: #f4f6f9; padding: 40px; transition: all 0.3s; }
.content-wrapper { max-width: 1500px; margin: 0 auto; display: flex; flex-direction: column; gap: 25px; }
.page-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 15px; }
.header-left h1 { margin: 0; font-size: 28px; color: #303133; transition: color 0.3s; }
.subtitle { margin: 8px 0 0; color: #909399; font-size: 16px; transition: color 0.3s; }
.menu-layout { display: flex; background: #fff; box-shadow: 0 4px 16px 0 rgba(0,0,0,0.08); border-radius: 10px; min-height: 800px; }
:deep(.el-tabs__header.is-left) { width: 260px; margin-right: 0; background: #fafafa; border-right: 1px solid #ebeef5; border-radius: 10px 0 0 10px; padding-top: 20px; }
:deep(.el-tabs__item.is-active) { background: #e6f7ff; color: #409EFF; border-right: 4px solid #409EFF !important; font-weight: bold; }
.menu-item { display: flex; align-items: center; gap: 12px; }
.menu-item .el-icon { font-size: 22px; }
:deep(.el-tabs__content) { flex: 1; padding: 30px; background: #fff; border-radius: 0 10px 10px 0; }
.card-header-flex { display: flex; justify-content: space-between; align-items: center; }
.card-title { font-weight: bold; font-size: 18px; border-left: 4px solid #409EFF; padding-left: 12px; }
.stat-cards { display: grid; grid-template-columns: repeat(3, 1fr); gap: 25px; }
.stat-item :deep(.el-card__body) { display: flex; align-items: center; padding: 25px; }
.stat-icon { width: 70px; height: 70px; border-radius: 14px; display: flex; align-items: center; justify-content: center; font-size: 36px; color: #fff; margin-right: 25px; }
.bg-blue { background: linear-gradient(135deg, #36d1dc, #5b86e5); }
.bg-green { background: linear-gradient(135deg, #11998e, #38ef7d); }
.bg-purple { background: linear-gradient(135deg, #c471ed, #f64f59); }
.stat-info { display: flex; flex-direction: column; }
.stat-value { font-size: 34px; font-weight: bold; color: #303133; }
.stat-label { font-size: 15px; color: #909399; margin-top: 5px; }
.chart-row { display: grid; grid-template-columns: 1fr 1fr; gap: 25px; margin-top: 25px; }
.chart-card { height: 450px; display: flex; flex-direction: column; }
.chart-box { height: 380px; width: 100%; }
.table-card { min-height: 500px; border: none; }
:deep(.el-table) { font-size: 15px; }
.user-cell { display: flex; align-items: center; gap: 10px; }
:deep(.large-dialog .el-dialog__title) { font-size: 20px; font-weight: bold; }
:deep(.large-dialog .el-form-item__label) { font-size: 15px; }

/* 聊天气泡（公用） */
.feedback-layout { display: flex; height: 700px; gap: 25px; }
.chat-context-panel { flex: 1.5; border: 1px solid #e0e0e0; border-radius: 8px; display: flex; flex-direction: column; background: #f9f9f9; }
.panel-header { padding: 15px 20px; background: #fff; border-bottom: 1px solid #eee; font-size: 16px; font-weight: bold; color: #333; border-radius: 8px 8px 0 0; }
.chat-window { flex: 1; overflow-y: auto; padding: 20px; display: flex; flex-direction: column; gap: 20px; }
.chat-bubble-row { display: flex; flex-direction: column; gap: 8px; position: relative; }
.chat-bubble-row.is-target { background: rgba(245, 108, 108, 0.08); border: 1px dashed #F56C6C; padding: 15px; border-radius: 10px; }
.target-mark { position: absolute; right: 15px; top: 0%; transform: translateY(-50%); color: #F56C6C; font-weight: bold; font-size: 13px; background: #fff; padding: 0 5px; }
.bubble { max-width: 90%; padding: 10px 15px; border-radius: 10px; font-size: 15px; line-height: 1.6; }
.bubble.user { align-self: flex-end; background: #409EFF; color: #fff; }
.bubble.ai { align-self: flex-start; background: #fff; border: 1px solid #ddd; color: #333; }
.role { font-size: 13px; opacity: 0.8; margin-bottom: 4px; font-weight: bold; }

.feedback-action-panel { flex: 1; display: flex; flex-direction: column; gap: 25px; }
.info-block label { display: block; font-size: 14px; color: #999; margin-bottom: 8px; }
.info-block .val { font-size: 16px; color: #333; font-weight: 500; }
.info-block .val.reason { color: #E6A23C; background: #fdf6ec; padding: 15px; border-radius: 6px; line-height: 1.6; }
.action-footer { margin-top: auto; display: flex; justify-content: flex-end; gap: 15px; }

html.dark .admin-container { background-color: transparent; }
html.dark .header-left h1, html.dark .subtitle, html.dark .stat-value, html.dark .card-title { color: #E0E0E0; }
html.dark .menu-layout { background-color: rgba(30,30,30,0.8); border: none; }
html.dark :deep(.el-tabs__header.is-left) { background: rgba(20,20,20,0.8); border-right: 1px solid rgba(255,255,255,0.1); }
html.dark :deep(.el-tabs__item) { color: #bbb; }
html.dark :deep(.el-tabs__item:hover) { background: rgba(255,255,255,0.05); }
html.dark :deep(.el-tabs__item.is-active) { background: rgba(64,158,255,0.15); color: #409EFF; border-right-color: #409EFF !important; }
html.dark :deep(.el-tabs__content) { background: transparent; }
html.dark .el-card { background-color: rgba(30, 30, 30, 0.6); backdrop-filter: blur(10px); border-color: rgba(255, 255, 255, 0.1); color: #E0E0E0; }
html.dark .el-table { background-color: transparent; color: #E0E0E0; --el-table-tr-bg-color: transparent; --el-table-header-bg-color: rgba(255,255,255,0.05); }
html.dark .chat-context-panel { background: rgba(0,0,0,0.3); border-color: #444; }
html.dark .panel-header { background: rgba(255,255,255,0.05); color: #ccc; border-bottom-color: #444; }
html.dark .bubble.ai { background: #333; color: #eee; border-color: #555; }
html.dark .info-block .val { color: #ddd; }
html.dark .info-block .val.reason { background: rgba(230, 162, 60, 0.1); color: #E6A23C; }
</style>