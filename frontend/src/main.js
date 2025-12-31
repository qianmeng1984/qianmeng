import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'

// 1. 引入 Element Plus 和 图标
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
// 引入黑夜模式样式
import 'element-plus/theme-chalk/dark/css-vars.css'
import * as ElementPlusIconsVue from '@element-plus/icons-vue'

// import './assets/main.css' // 保持默认样式或之后清空

const app = createApp(App)

// 2. 注册所有图标
for (const [key, component] of Object.entries(ElementPlusIconsVue)) {
    app.component(key, component)
}

app.use(createPinia())
app.use(router)
app.use(ElementPlus) // 使用 UI 库

app.mount('#app')