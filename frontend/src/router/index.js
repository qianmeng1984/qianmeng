import { createRouter, createWebHistory } from 'vue-router'
import Login from '../views/Login.vue'
import Chat from '../views/Chat.vue'
import Knowledge from '../views/Knowledge.vue'
import Profile from '../views/Profile.vue'
import Admin from '../views/Admin.vue'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    { path: '/login', name: 'login', component: Login },
    // 主页改成 Chat
    { path: '/', name: 'chat', component: Chat },
    // 知识库页面
    { path: '/knowledge', name: 'knowledge', component: Knowledge },
    { path: '/profile', name: 'profile', component: Profile },
    { path: '/admin', component: Admin }, // 新增这行
  ]
})

// 简单的路由守卫：没 Token 不让进主页
router.beforeEach((to, from, next) => {
  const token = localStorage.getItem('token')
  if (to.name !== 'login' && !token) {
    next({ name: 'login' })
  } else {
    next()
  }
})

export default router