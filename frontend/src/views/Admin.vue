<template>
  <div class="admin-container">
    <el-card class="admin-card">
      <template #header>
        <div class="card-header">
          <span>🛡️ 管理员控制台</span>
          <el-button @click="router.back()">返回对话</el-button>
        </div>
      </template>

      <el-table :data="userList" stripe style="width: 100%" v-loading="loading">
        <el-table-column prop="ID" label="ID" width="80" />
        <el-table-column label="头像" width="80">
          <template #default="scope">
            <el-avatar :size="30" :src="getFullUrl(scope.row.Avatar)" />
          </template>
        </el-table-column>
        <el-table-column prop="Username" label="账号" />
        <el-table-column prop="Nickname" label="昵称" />
        <el-table-column label="角色" width="100">
          <template #default="scope">
            <el-tag :type="scope.row.Role === 1 ? 'danger' : 'info'">
              {{ scope.row.Role === 1 ? '管理员' : '用户' }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column label="操作" align="right">
          <template #default="scope">
            <el-button size="small" @click="openEdit(scope.row)">编辑</el-button>
            <el-popconfirm title="确定注销？" @confirm="handleDelete(scope.row.ID)">
              <template #reference>
                <el-button size="small" type="danger" :disabled="scope.row.Role === 1">注销</el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog v-model="dialogVisible" title="修改用户信息" width="400px">
      <el-form :model="editForm" label-width="80px">
        <el-form-item label="账号"><el-input v-model="editForm.username" disabled /></el-form-item>
        <el-form-item label="昵称"><el-input v-model="editForm.nickname" /></el-form-item>
        <el-form-item label="重置密码"><el-input v-model="editForm.password" placeholder="留空不改" show-password /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleUpdate">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import request from '@/utils/request'
import { ElMessage } from 'element-plus'

const router = useRouter()
const userList = ref([])
const loading = ref(false)
const dialogVisible = ref(false)
const editForm = ref({ id: 0, username: '', nickname: '', password: '' })
const BASE_URL = 'http://localhost:8080'
const getFullUrl = (path) => path ? (path.startsWith('http') ? path : BASE_URL + path) : ''

const fetchUsers = async () => {
  loading.value = true
  try {
    const res = await request.get('/admin/users')
    userList.value = res || []
  } catch (e) { router.push('/chat') } finally { loading.value = false }
}

const handleDelete = async (id) => {
  try { await request.post('/admin/user/delete', { id }); ElMessage.success('已注销'); fetchUsers() } catch(e){}
}
const openEdit = (row) => {
  editForm.value = { id: row.ID, username: row.Username, nickname: row.Nickname, password: '' }
  dialogVisible.value = true
}
const handleUpdate = async () => {
  try {
    await request.post('/admin/user/update', editForm.value)
    ElMessage.success('修改成功'); dialogVisible.value = false; fetchUsers()
  } catch(e) { ElMessage.error('失败') }
}
onMounted(() => { fetchUsers() })
</script>

<style scoped>
/* 同样去除背景色 */
.admin-container {
  padding: 40px;
  display: flex;
  justify-content: center;
  min-height: 100vh;
  /* background: #f4f4f7; <-- 删除这行 */
}
.admin-card { width: 900px; height: fit-content; }
.card-header { display: flex; justify-content: space-between; align-items: center; font-weight: bold; }
</style>