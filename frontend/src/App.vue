<template>
  <router-view />
</template>

<script setup>
import { onMounted } from 'vue'

// 全局初始化主题
onMounted(() => {
  const savedTheme = localStorage.getItem('token') ? localStorage.getItem('theme') : 'light'
  if (savedTheme === 'dark') {
    document.documentElement.classList.add('dark')
  } else {
    document.documentElement.classList.remove('dark')
  }
})
</script>

<style>
/* 全局样式重置 */
body {
  margin: 0;
  padding: 0;
  font-family: 'Helvetica Neue', Helvetica, 'PingFang SC', 'Hiragino Sans GB', 'Microsoft YaHei', '微软雅黑', Arial, sans-serif;
}

/* ========================================= */
/* 🌌 全局星空黑夜模式特效 (Global Starry Sky) */
/* ========================================= */

/* 当处于黑夜模式时，背景变成深空蓝 */
html.dark body {
  background: linear-gradient(to bottom, #0f2027, #203a43, #2c5364); /* 深邃星空渐变 */
  background-attachment: fixed; /* 固定背景，防止滚动时断层 */
  min-height: 100vh;
  color: #e0e0e0;
}

/* 绘制星星 - 使用 CSS 径向渐变模拟 */
html.dark body::before {
  content: "";
  position: fixed;
  top: 0; left: 0; width: 100%; height: 100%;
  pointer-events: none; /* 让星星不挡住点击 */
  background-image:
      radial-gradient(2px 2px at 20px 30px, #ffffff, rgba(0,0,0,0)),
      radial-gradient(2px 2px at 40px 70px, #ffffff, rgba(0,0,0,0)),
      radial-gradient(2px 2px at 50px 160px, #ffffff, rgba(0,0,0,0)),
      radial-gradient(2px 2px at 90px 40px, #ffffff, rgba(0,0,0,0)),
      radial-gradient(2px 2px at 130px 80px, #ffffff, rgba(0,0,0,0)),
      radial-gradient(2px 2px at 160px 120px, #ffffff, rgba(0,0,0,0));
  background-repeat: repeat;
  background-size: 200px 200px;
  opacity: 0.3;
  z-index: -1;
  animation: twinkle 5s infinite linear;
}

/* 星星闪烁动画 */
@keyframes twinkle {
  0% { transform: translateY(0); }
  100% { transform: translateY(-200px); }
}

/* 强制让 Element Plus 的卡片在黑夜模式下半透明，透出星空 */
html.dark .el-card {
  background-color: rgba(30, 30, 30, 0.8) !important; /* 半透明黑 */
  border-color: #444 !important;
  color: #fff !important;
  backdrop-filter: blur(10px); /* 毛玻璃特效 */
}

/* 表格黑夜模式适配 */
html.dark .el-table {
  background-color: transparent !important;
  color: #ddd !important;
  --el-table-tr-bg-color: transparent !important;
  --el-table-header-bg-color: rgba(50, 50, 50, 0.5) !important;
}
html.dark .el-table tr {
  background-color: transparent !important;
}
html.dark .el-table--striped .el-table__body tr.el-table__row--striped td.el-table__cell {
  background-color: rgba(255, 255, 255, 0.05) !important;
}
html.dark .el-table td.el-table__cell,
html.dark .el-table th.el-table__cell.is-leaf {
  border-bottom: 1px solid #444 !important;
}
</style>