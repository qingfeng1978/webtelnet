<template>
  <div class="app-container" :class="{ 'dark-theme': isDarkTheme }">
    <el-container>
      <el-aside width="260px" class="server-panel">
        <div class="panel-header">
          <h2>服务器列表</h2>
          <el-button
            class="theme-toggle"
            circle
            @click="toggleTheme"
          >
            <el-icon v-if="isDarkTheme"><Sunny /></el-icon>
            <el-icon v-else><Moon /></el-icon>
          </el-button>
        </div>
        <ServerTree @select-server="handleSelectServer" />
      </el-aside>
      <el-main class="main-content">
        <div v-if="hasTerminals" class="terminals-container">
          <div class="terminal-toolbar">
            <el-tooltip content="搜索终端内容 (Ctrl+F)" placement="bottom">
              <el-button
                size="small"
                circle
                @click="searchTerminal"
                class="terminal-action-btn"
              >
                <el-icon><Search /></el-icon>
              </el-button>
            </el-tooltip>
          </div>
          <TerminalManager 
            ref="terminalManager"
            @terminals-changed="handleTerminalsChange"
            @terminal-created="handleTerminalCreated"
          />
        </div>
        <div v-else class="welcome-message">
          <h2>WebTelnet 客户端</h2>
          <p>从左侧选择服务器连接</p>
        </div>
      </el-main>
    </el-container>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, nextTick } from 'vue'
import ServerTree from './components/ServerTree.vue'
import TerminalManager from './components/TerminalManager.vue'
import { ElMessage } from 'element-plus'
import { Moon, Sunny, Search } from '@element-plus/icons-vue'
import 'element-plus/dist/index.css'
import type { ComponentPublicInstance } from 'vue'

// 定义TerminalManager组件实例的类型接口
interface TerminalManagerInstance extends ComponentPublicInstance {
  createNewTerminal: (serverInfo?: any) => any;
  closeTerminal: (targetId: string) => void;
  cleanupAllTerminals: () => void;
  checkAndFixTerminalRefs: () => void;
  terminals: any[];
  activeTab: string;
  showSearch: () => void;
  hideSearch: () => void;
}

const form = ref({
  protocol: 'telnet',
  host: 'localhost',
  port: 23,
  username: '',
  password: ''
})

// 添加类型定义，从any改为具体类型
const terminalManager = ref<TerminalManagerInstance | null>(null)
const hasTerminals = ref(false)
const pendingConnections = ref<any[]>([]) // 存储待处理的连接请求

// 添加主题相关代码
const isDarkTheme = ref(localStorage.getItem('theme') === 'dark')

const toggleTheme = () => {
  isDarkTheme.value = !isDarkTheme.value
  localStorage.setItem('theme', isDarkTheme.value ? 'dark' : 'light')
  document.documentElement.setAttribute('data-theme', isDarkTheme.value ? 'dark' : 'light')
  
  // 触发自定义事件，通知其他组件主题已变化
  window.dispatchEvent(new CustomEvent('theme-change', { 
    detail: { theme: isDarkTheme.value ? 'dark' : 'light' } 
  }))
}

// 处理服务器选择事件
const handleSelectServer = (server: any) => {
  console.log('服务器选择事件触发:', {
    ...server, 
    password: server.password ? '******' : '<未设置>'
  })
  
  form.value.host = server.host
  form.value.port = server.port
  form.value.protocol = server.port === 22 ? 'ssh' : 'telnet'
  form.value.username = server.username || ''
  form.value.password = server.password || ''
  
  // 如果是SSH协议且有用户名密码，或者是Telnet协议且有用户名密码，则自动连接
  if ((form.value.protocol === 'ssh' && form.value.username) || 
      (form.value.protocol === 'telnet' && form.value.username)) {
    
    // 检查terminalManager引用是否存在
    console.log('TerminalManager引用状态:', 
      terminalManager.value ? '已引用' : '未引用',
      terminalManager.value ? `(类型: ${typeof terminalManager.value})` : '')
    
    // 通过引用调用TerminalManager的方法
    if (terminalManager.value) {
      try {
        // 检查TerminalManager上可用的方法
        const managerMethods = {
          createNewTerminal: typeof terminalManager.value.createNewTerminal === 'function',
          closeTerminal: typeof terminalManager.value.closeTerminal === 'function',
          cleanupAllTerminals: typeof terminalManager.value.cleanupAllTerminals === 'function'
        }
        console.log('TerminalManager可用方法:', 
          managerMethods.createNewTerminal ? '√ createNewTerminal' : '✗ createNewTerminal',
          managerMethods.closeTerminal ? '√ closeTerminal' : '✗ closeTerminal',
          managerMethods.cleanupAllTerminals ? '√ cleanupAllTerminals' : '✗ cleanupAllTerminals')
        
        // 解决hasTerminals初始为false时的问题
        if (!hasTerminals.value) {
          console.log('终端容器未显示，设置hasTerminals为true并延迟创建终端')
          hasTerminals.value = true
          
          // 将连接请求添加到队列
          pendingConnections.value.push(server)
          
          // 在下一个渲染周期执行连接，确保TerminalManager已渲染
          nextTick(() => {
            processPendingConnections()
          })
        } else {
          // 如果已经显示终端容器，直接创建终端
          createTerminalForServer(server)
        }
      } catch (error) {
        console.error('创建终端失败:', error)
        ElMessage.error(`连接失败: ${error instanceof Error ? error.message : '未知错误'}`)
      }
    } else {
      console.error('TerminalManager组件引用不存在!')
      
      // 将服务器添加到待处理队列并显示终端容器
      hasTerminals.value = true
      pendingConnections.value.push(server)
      
      ElMessage.info('正在准备连接，请稍候...')
    }
  } else {
    console.log('缺少用户名或密码，不自动连接')
    ElMessage.info('请编辑服务器添加用户名和密码以自动连接')
  }
}

// 处理终端容器的TerminalManager组件创建成功事件
const handleTerminalCreated = () => {
  console.log('TerminalManager组件创建成功通知')
  
  // 延迟处理，确保DOM更新完成
  setTimeout(() => {
    processPendingConnections()
  }, 100)
}

// 处理待处理的连接请求
const processPendingConnections = () => {
  if (pendingConnections.value.length > 0 && terminalManager.value) {
    console.log(`处理${pendingConnections.value.length}个待处理的连接请求`)
    
    // 复制并清空待处理队列
    const servers = [...pendingConnections.value]
    pendingConnections.value = []
    
    // 为每个服务器创建终端
    servers.forEach(server => {
      createTerminalForServer(server)
    })
  }
}

// 创建终端连接
const createTerminalForServer = (server: any) => {
  if (terminalManager.value) {
    console.log('创建终端连接:', server.name)
    const newTerminal = terminalManager.value.createNewTerminal(server)
    console.log('终端创建成功:', newTerminal)
    ElMessage.success(`正在${newTerminal.protocol === 'ssh' ? 'SSH' : 'Telnet'}连接到 ${server.name}`)
  }
}

// 处理终端数量变化事件
const handleTerminalsChange = (count: number) => {
  console.log('终端数量变化:', count)
  hasTerminals.value = count > 0
}

// 搜索终端内容
const searchTerminal = () => {
  if (terminalManager.value) {
    try {
      terminalManager.value.showSearch()
    } catch (error) {
      console.error('调用终端搜索失败:', error)
      ElMessage.error('无法启动搜索功能')
    }
  } else {
    console.error('终端管理器引用不存在')
  }
}

// 初始化主题
onMounted(() => {
  const theme = localStorage.getItem('theme') || 'light'
  isDarkTheme.value = theme === 'dark'
  document.documentElement.setAttribute('data-theme', theme)
  
  // 添加调试日志，验证terminalManager组件是否正确引用
  console.log('App.vue已挂载，TerminalManager组件引用状态:', terminalManager.value)
  
  // 检查组件在下一个渲染周期后是否可用
  setTimeout(() => {
    console.log('延迟检查: TerminalManager组件引用状态:', terminalManager.value)
    
    // 处理任何待处理的连接请求
    processPendingConnections()
  }, 500)
  
  // 添加页面卸载事件监听
  window.addEventListener('beforeunload', handleBeforeUnload)
})

onUnmounted(() => {
  console.log('App.vue已卸载')
  window.removeEventListener('beforeunload', handleBeforeUnload)
})

// 页面卸载前清理资源
const handleBeforeUnload = () => {
  console.log('页面即将卸载，清理资源')
  if (terminalManager.value) {
    terminalManager.value.cleanupAllTerminals()
  }
}
</script>

<style>
/* 添加主题变量 */
:root {
  --bg-color: #ffffff;
  --text-color: #333333;
  --border-color: #e4e7ed;
  --hover-color: #f5f7fa;
  --active-color: #ecf5ff;
  --primary-color: #409eff;
  --panel-bg: #f5f7fa;
  --terminal-bg: #1e1e1e;
  --tooltip-bg-light: #fff;
  --tooltip-text-light: #5a5a5a;
  --tooltip-bg-dark: #303133;
  --tooltip-text-dark: #fff;
}

:root[data-theme="dark"] {
  --bg-color: #1a1a1a;
  --text-color: #ffffff;
  --border-color: #4c4d4f;
  --hover-color: #2c2c2c;
  --active-color: #2c3e50;
  --primary-color: #409eff;
  --panel-bg: #2b2b2b;
  --terminal-bg: #1e1e1e;
}

:root[data-theme="dark"] .el-popper.is-light {
  background: var(--tooltip-bg-dark) !important;
  color: var(--tooltip-text-dark) !important;
  border-color: var(--border-color) !important;
}

:root[data-theme="dark"] .el-popper.is-light .el-popper__arrow::before {
  background: var(--tooltip-bg-dark) !important;
  border-color: var(--border-color) !important;
}

/* 重置默认样式 */
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

html, body {
  height: 100%;
  width: 100%;
  overflow: hidden;
  margin: 0;
  padding: 0;
  background-color: var(--bg-color);
  color: var(--text-color);
}

.app-container {
  height: 100vh;
  width: 100vw;
  overflow: hidden;
  margin: 0;
  padding: 0;
  display: flex;
  position: fixed;
  left: 0;
  top: 0;
  background-color: var(--bg-color);
}

.el-container {
  width: 100%;
  height: 100%;
  margin: 0 !important;
  padding: 0 !important;
  display: flex;
}

/* 侧边栏样式 */
.server-panel {
  background-color: var(--panel-bg);
  border-right: 1px solid var(--border-color);
  display: flex;
  flex-direction: column;
  height: 100%;
  color: var(--text-color);
  padding: 0;
  width: 260px !important;
  margin: 0;
  flex-shrink: 0;
  position: relative;
  left: 0;
}

/* 覆盖 Element Plus 的默认样式 */
.el-aside {
  padding: 0 !important;
  margin: 0 !important;
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 14px 16px;
  border-bottom: 1px solid var(--border-color);
  background-color: var(--panel-bg);
  color: var(--text-color);
}

.panel-header h2 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: var(--text-color);
}

.main-content {
  height: 100%;
  padding: 0 !important;
  background-color: var(--bg-color);
  color: var(--text-color);
  overflow: hidden;
  position: relative;
}

.terminals-container {
  height: 100%;
  width: 100%;
  position: relative;
}

.welcome-message {
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  height: 100%;
  text-align: center;
  color: var(--text-color);
}

.welcome-message h2 {
  font-size: 28px;
  margin-bottom: 16px;
}

.welcome-message p {
  font-size: 18px;
  opacity: 0.7;
}

/* 主题切换按钮样式 */
.theme-toggle {
  background: transparent;
  border: none;
  cursor: pointer;
  font-size: 20px;
  color: var(--text-color);
  transition: transform 0.3s ease;
}

.theme-toggle:hover {
  transform: rotate(30deg);
}

.dark-theme {
  color-scheme: dark;
}

/* 终端工具栏 */
.terminal-toolbar {
  position: absolute;
  top: 10px;
  right: 10px;
  z-index: 10;
  display: flex;
  gap: 8px;
}

.terminal-action-btn {
  background-color: var(--bg-color);
  border-color: var(--border-color);
  color: var(--text-color);
  opacity: 0.6;
  transition: opacity 0.3s ease;
}

.terminal-action-btn:hover {
  opacity: 1;
  background-color: var(--hover-color);
}
</style>

