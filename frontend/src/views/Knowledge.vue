<template>
  <div class="knowledge-container">
    <div class="content-wrapper">
      <el-card class="box-card upload-section">
        <template #header>
          <div class="card-header">
            <span>📚 上传新文档</span>
            <el-button @click="router.back()">返回对话</el-button>
          </div>
        </template>

        <div class="upload-area">
          <el-upload
              class="upload-demo"
              drag
              :action="uploadUrl"
              :headers="headers"
              :data="{ public: false }"
              accept=".txt,.md"
              :on-success="handleSuccess"
              :on-error="handleError"
              :show-file-list="false"
              multiple
          >
            <el-icon class="el-icon--upload"><UploadFilled /></el-icon>
            <div class="el-upload__text">
              拖拽 .txt / .md  文件到此处<br><em>(上传后自动切片并存入向量库)</em>
            </div>
          </el-upload>
        </div>
      </el-card>

      <el-card class="box-card list-section">
        <template #header>
          <div class="card-header">
            <span>💾 已入库文件管理</span>
            <el-button link type="primary" @click="fetchFileList">
              <el-icon><Refresh /></el-icon> 刷新
            </el-button>
          </div>
        </template>

        <el-table :data="fileList" style="width: 100%" stripe v-loading="loading">
          <el-table-column label="文件名" prop="name">
            <template #default="scope">
              <el-icon style="vertical-align: middle; margin-right: 5px"><Document /></el-icon>
              {{ scope.row }}
            </template>
          </el-table-column>

          <el-table-column label="操作" width="120" align="right">
            <template #default="scope">
              <el-popconfirm
                  title="确定删除？"
                  confirm-button-text="删除"
                  cancel-button-text="取消"
                  @confirm="handleDelete(scope.row)"
              >
                <template #reference>
                  <el-button type="danger" size="small" :icon="Delete" circle />
                </template>
              </el-popconfirm>
            </template>
          </el-table-column>
        </el-table>
        <el-empty v-if="fileList.length === 0" description="暂无上传记录" />
      </el-card>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { UploadFilled, Document, Delete, Refresh } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import request from '@/utils/request'

const router = useRouter()
const uploadUrl = 'http://localhost:8080/api/v1/upload'
const fileList = ref([])
const loading = ref(false)
const headers = computed(() => ({ Authorization: `Bearer ${localStorage.getItem('token')}` }))

const fetchFileList = async () => {
  loading.value = true
  try {
    const res = await request.get('/knowledge/list')
    fileList.value = res || []
  } catch (e) {} finally { loading.value = false }
}

const handleDelete = async (fileName) => {
  try {
    await request.post('/knowledge/delete', { file_name: fileName })
    ElMessage.success(`文件删除成功`)
    fetchFileList()
  } catch (e) { ElMessage.error('删除失败') }
}

const handleSuccess = () => { ElMessage.success('上传成功'); fetchFileList() }
const handleError = () => { ElMessage.error('上传失败') }
onMounted(() => { fetchFileList() })
</script>

<style scoped>
/* 关键修改：背景设为透明，以便透出 App.vue 定义的星空 */
.knowledge-container {
  padding: 40px;
  min-height: 100vh;
  display: flex;
  justify-content: center;
  /* background: #f4f4f7;  <-- 删除这行，不要覆盖全局背景 */
}

.content-wrapper {
  width: 800px;
  display: flex;
  flex-direction: column;
  gap: 20px;
}
.card-header { display: flex; justify-content: space-between; align-items: center; font-weight: bold; }
.upload-area { padding: 20px 0; }
.list-section { min-height: 300px; }
</style>