import { createApp } from 'vue'
import './style.css'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import App from './App.vue'

// 创建Vue应用
const app = createApp(App)

// 添加一个全局错误处理器以便捕获和记录错误
app.config.errorHandler = (err, vm, info) => {
  console.error('Vue错误:', err)
  console.error('错误组件:', vm)
  console.error('错误信息:', info)
}

// 使用ElementPlus
app.use(ElementPlus)

// 挂载应用
app.mount('#app')

// 为调试目的添加到window对象
// 解决TypeScript错误，使用声明来扩展Window接口
declare global {
  interface Window {
    app: any;
  }
}
window.app = app
