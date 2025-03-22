<template>
  <div class="terminals-container">
    <el-tabs 
      v-model="activeTab" 
      type="card" 
      closable 
      @tab-remove="closeTerminal"
      @tab-click="handleTabChange"
    >
      <el-tab-pane
        v-for="term in terminals"
        :key="term.id"
        :label="term.title"
        :name="term.id"
      >
        <div class="terminal-wrapper">
          <keep-alive>
            <Terminal 
              :host="term.host" 
              :port="term.port"
              :protocol="term.protocol"
              :username="term.username"
              :password="term.password"
              :id="term.sessionID"
              :ref="el => { 
                if (el) {
                  terminalRefs[term.id] = el;
                  sessionStore[term.sessionID] = el;
                }
              }"
            />
          </keep-alive>
          <!-- 搜索组件 -->
          <TerminalSearch
            v-if="showSearch && activeTab === term.id"
            :activeTerminalId="term.id"
            :terminalRef="terminalRefs[term.id]"
            :isDarkTheme="isDarkTheme"
            @close="hideSearch"
            @search-result="handleSearchResult"
            ref="searchComponent"
          />
        </div>
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, onUnmounted, defineEmits, defineExpose, watch, computed } from 'vue'
import Terminal from './Terminal.vue'
import TerminalSearch from './TerminalSearch.vue'
import { ElMessage } from 'element-plus'

interface TerminalInstance {
  id: string
  sessionID: string
  title: string
  host: string
  port: number
  protocol: string
  username: string
  password: string
  serverName?: string
}

const terminals = ref<TerminalInstance[]>([])
const terminalRefs = reactive<Record<string, any>>({})
const activeTab = ref('')
const sessionStore = reactive<Record<string, any>>({})

// 搜索相关状态
const showSearch = ref(false)
const searchComponent = ref<any>(null)
const searchResults = ref({ count: 0, current: -1 })
const isDarkTheme = ref(false)

// 提供事件以通知父组件状态变化
const emit = defineEmits(['tab-change', 'terminals-changed', 'terminal-created'])

// 处理主题变化
const updateTheme = (event: any) => {
  const theme = event.detail?.theme || localStorage.getItem('theme') || 'light'
  isDarkTheme.value = theme === 'dark'
}

const handleSearchResult = (result: { count: number, current: number }) => {
  searchResults.value = result
}

// 显示搜索组件
const showSearchBox = () => {
  if (terminals.value.length === 0) {
    ElMessage.warning('没有打开的终端')
    return
  }
  
  showSearch.value = true
  
  // 延迟聚焦搜索框
  setTimeout(() => {
    if (searchComponent.value) {
      searchComponent.value.focus()
    }
  }, 50)
}

// 隐藏搜索组件
const hideSearch = () => {
  showSearch.value = false
  
  // 清除搜索结果
  if (searchComponent.value) {
    searchComponent.value.clear()
  }
  
  // 聚焦当前终端
  if (activeTab.value && terminalRefs[activeTab.value]) {
    try {
      terminalRefs[activeTab.value].focusTerminal()
    } catch (error) {
      console.error('聚焦终端失败:', error)
    }
  }
}

// 处理键盘事件 - 搜索快捷键
const handleKeyDown = (event: KeyboardEvent) => {
  // Ctrl+F 或 Cmd+F: 打开搜索
  if ((event.ctrlKey || event.metaKey) && event.key === 'f') {
    event.preventDefault() // 阻止浏览器默认搜索行为
    showSearchBox()
  }
  
  // Esc: 关闭搜索
  if (event.key === 'Escape' && showSearch.value) {
    event.preventDefault()
    hideSearch()
  }
}

const generateId = () => {
  const id = `terminal_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`
  const sessionID = `session_${Date.now()}_${Math.random().toString(36).slice(2, 8)}`
  return { id, sessionID }
}

// 创建新终端实例
const createNewTerminal = (serverInfo: any = null) => {
  console.log('TerminalManager.createNewTerminal被调用，参数:', JSON.stringify({
    ...serverInfo,
    password: serverInfo?.password ? '******' : '' // 安全处理，不记录密码
  }))
  
  if (!serverInfo) {
    console.warn('创建终端时没有提供服务器信息')
    ElMessage.warning('没有提供服务器信息，将创建空白终端')
  }
  
  try {
    const { id, sessionID } = generateId()
    const newTerminal: TerminalInstance = {
      id,
      sessionID,
      title: serverInfo?.name || '新终端',
      host: serverInfo?.host || '',
      port: serverInfo?.port || 0,
      protocol: serverInfo?.port === 22 ? 'ssh' : 'telnet',
      username: serverInfo?.username || '',
      password: serverInfo?.password || '',
      serverName: serverInfo?.name
    }
    
    console.log(`正在添加新终端 ID:${id}, SessionID:${sessionID}, 标题:${newTerminal.title}`)
    terminals.value.push(newTerminal)
    activeTab.value = newTerminal.id
    
    // 通知父组件终端数量变化
    emit('terminals-changed', terminals.value.length)
    
    // 确保在下一个渲染周期更新引用
    setTimeout(() => {
      if (terminalRefs[id]) {
        console.log(`终端 ${id} 引用已设置成功`)
      } else {
        console.warn(`终端 ${id} 引用还未设置，可能在下一个渲染周期更新`)
      }
    }, 100)
    
    return newTerminal
  } catch (error) {
    console.error('创建新终端实例时发生错误:', error)
    ElMessage.error('创建终端失败，请重试')
    throw error
  }
}

const closeTerminal = (targetId: string) => {
  const index = terminals.value.findIndex(t => t.id === targetId)
  if (index !== -1) {
    console.log(`开始关闭终端 ${targetId}`)
    
    // 获取终端实例
    const terminal = terminalRefs[targetId]
    if (terminal) {
      try {
        // 断开连接并销毁终端资源
        console.log(`正在断开终端 ${targetId} 的连接并清理资源`)
        terminal.disconnect()
      } catch (error) {
        console.error(`断开终端 ${targetId} 连接时出错:`, error)
        // 即使断开失败，仍继续关闭流程
      }
    } else {
      console.warn(`找不到终端 ${targetId} 的引用`)
    }
    
    // 删除终端引用，无论断开是否成功
    delete terminalRefs[targetId]
    console.log(`终端 ${targetId} 引用已删除`)
    
    // 安全从会话存储中删除
    try {
      const sessionId = terminals.value[index].sessionID
      if (sessionId && sessionStore[sessionId]) {
        delete sessionStore[sessionId]
        console.log(`会话 ${sessionId} 已从存储中移除`)
      }
    } catch (sessionError) {
      console.error('清理会话存储时出错:', sessionError)
    }
    
    // 从数组中移除终端
    terminals.value.splice(index, 1)
    console.log(`终端 ${targetId} 已从列表中移除`)
    
    // 通知父组件终端数量变化
    emit('terminals-changed', terminals.value.length)
    
    // 如果还有其他终端，切换到最后一个
    if (terminals.value.length > 0) {
      const newActiveId = terminals.value[terminals.value.length - 1].id
      console.log(`切换到终端 ${newActiveId}`)
      activeTab.value = newActiveId
      
      // 确保新活跃终端被激活
      setTimeout(() => {
        const newActiveTerminal = terminalRefs[newActiveId]
        if (newActiveTerminal) {
          try {
            console.log(`激活终端 ${newActiveId}`)
            newActiveTerminal.isTerminalActive = true
            
            // 触发调整大小
            setTimeout(() => {
              window.dispatchEvent(new Event('resize'))
            }, 100)
          } catch (error) {
            console.error(`激活终端 ${newActiveId} 失败:`, error)
            // 即使激活失败，用户仍可以看到终端内容
          }
        }
      }, 50)
    }
    
    ElMessage.success('终端已关闭')
  } else {
    console.warn(`找不到ID为 ${targetId} 的终端`)
  }
}

const handleTabChange = (tab: any) => {
  try {
    // 提取标签ID
    const tabId = tab.props.name
    console.log('切换到标签:', tabId)
    
    // 将状态发送到父组件
    emit('tab-change', tabId)
    
    // 延迟一下，确保TabPane的样式已经更新
    setTimeout(() => {
      // 所有终端保持活跃状态，只是改变UI显示
      for (const termId in terminalRefs) {
        const termRef = terminalRefs[termId]
        if (termRef) {
          try {
            if (termId === tabId) {
              // 更新UI显示，但保持WebSocket连接
              console.log(`更新 ${termId} 终端UI显示为可见`)
              
              // 设置终端为可见状态
              try {
                if (typeof termRef.isTerminalActive === 'undefined') {
                  console.warn(`终端 ${termId} 没有isTerminalActive属性，可能需要刷新页面`)
                } else {
                  termRef.isTerminalActive = true
                }
              } catch (activateError) {
                console.error(`激活终端 ${termId} 失败:`, activateError)
              }
              
              // 确保终端内容完全显示，多次尝试调整大小和刷新
              const maxResizeAttempts = 3;
              
              // 第一次尝试，较短延迟
              setTimeout(() => attemptResize(termId, termRef, 0), 100);
              
              // 第二次尝试，中等延迟
              setTimeout(() => attemptResize(termId, termRef, 1), 500);
              
              // 第三次尝试，较长延迟
              setTimeout(() => attemptResize(termId, termRef, 2), 1500);
              
              // 尝试调整终端大小和刷新
              function attemptResize(termId: string, termRef: any, attempt: number) {
                if (attempt >= maxResizeAttempts) return;
                
                console.log(`第${attempt + 1}次尝试调整 ${termId} 终端大小和刷新显示`);
                
                // 首先尝试调用终端的resize方法
                if (termRef.resize && typeof termRef.resize === 'function') {
                  try {
                    termRef.resize();
                    console.log(`已调用 ${termId} 终端的resize方法（尝试 ${attempt + 1}）`);
                  } catch (error) {
                    console.error(`调用 ${termId} 终端resize方法失败 (尝试 ${attempt + 1}):`, error);
                    
                    // 如果是最后一次尝试，触发一个全局resize事件作为后备方案
                    if (attempt === maxResizeAttempts - 1) {
                      window.dispatchEvent(new Event('resize'));
                    }
                  }
                } else {
                  console.warn(`终端 ${termId} 没有resize方法，使用全局resize事件`);
                  window.dispatchEvent(new Event('resize'));
                }
                
                // 尝试聚焦终端
                if (attempt === maxResizeAttempts - 1 && termRef.focusTerminal && typeof termRef.focusTerminal === 'function') {
                  try {
                    termRef.focusTerminal();
                    console.log(`已聚焦 ${termId} 终端 (最后尝试)`);
                  } catch (error) {
                    console.error(`聚焦 ${termId} 终端失败:`, error);
                  }
                }
              }
            } else {
              // 其他标签只需更新UI显示状态，但保持WebSocket连接和活跃状态
              console.log(`更新 ${termId} 终端UI显示为隐藏`)
              try {
                if (typeof termRef.isTerminalActive !== 'undefined') {
                  termRef.isTerminalActive = false
                }
              } catch (error) {
                console.error(`设置终端 ${termId} 为隐藏状态失败:`, error)
              }
            }
          } catch (error) {
            console.error(`处理终端 ${termId} UI显示状态失败:`, error)
          }
        } else {
          console.warn(`终端引用 ${termId} 为空`)
        }
      }
    }, 50) // 等待DOM更新
  } catch (error) {
    console.error('标签切换处理失败:', error)
  }
}

const cleanupAllTerminals = () => {
  console.log('清理所有终端资源...')
  
  // 遍历所有终端引用并断开连接
  for (const termId in terminalRefs) {
    try {
      const terminal = terminalRefs[termId]
      if (terminal) {
        console.log(`清理终端 ${termId} 资源`)
        terminal.disconnect()
      }
    } catch (error) {
      console.error(`清理终端 ${termId} 资源时出错:`, error)
    }
  }
  
  // 清空引用
  for (const termId in terminalRefs) {
    delete terminalRefs[termId]
  }
  console.log('所有终端资源已清理')
}

const checkAndFixTerminalRefs = () => {
  console.log('检查终端引用和会话状态...')
  
  // 检查每个终端是否有对应的引用
  for (const terminal of terminals.value) {
    if (!terminalRefs[terminal.id]) {
      console.warn(`终端 ${terminal.id} (sessionID: ${terminal.sessionID}) 引用丢失，尝试修复`)
      // 尝试通过sessionID恢复引用
      if (sessionStore[terminal.sessionID]) {
        console.log(`通过sessionID ${terminal.sessionID} 找到会话信息，正在恢复引用`)
        terminalRefs[terminal.id] = sessionStore[terminal.sessionID]
      } else {
        // 在下一个渲染周期尝试修复引用
        setTimeout(() => {
          if (!terminalRefs[terminal.id]) {
            console.error(`无法修复终端 ${terminal.id} 的引用，可能需要重新创建`)
          }
        }, 500)
      }
    } else {
      // 保存会话引用到会话存储
      sessionStore[terminal.sessionID] = terminalRefs[terminal.id]
    }
  }
  
  // 检查是否有多余的引用
  for (const refId in terminalRefs) {
    const found = terminals.value.some(term => term.id === refId)
    if (!found) {
      const sessionID = terminalRefs[refId]?.sessionID
      console.warn(`发现多余的终端引用 ${refId} (sessionID: ${sessionID})，删除`)
      delete terminalRefs[refId]
      if (sessionID) {
        delete sessionStore[sessionID]
      }
    }
  }
}

// 公开方法供父组件调用
defineExpose({
  createNewTerminal,
  closeTerminal,
  cleanupAllTerminals,
  checkAndFixTerminalRefs,
  terminals: terminals.value,
  activeTab,
  showSearch: showSearchBox,
  hideSearch
})

// 初始化和清理
let checkInterval: number | undefined;

onMounted(() => {
  console.log('TerminalManager组件已挂载')
  
  // 触发终端创建事件通知
  emit('terminal-created')
  
  // 添加键盘事件监听
  window.addEventListener('keydown', handleKeyDown)
  
  // 监听主题变化事件
  window.addEventListener('theme-change', updateTheme)
  
  // 初始化主题状态
  isDarkTheme.value = localStorage.getItem('theme') === 'dark'
})

onUnmounted(() => {
  console.log('TerminalManager组件即将卸载')
  
  // 卸载所有终端和会话
  cleanupAllTerminals()
  
  // 移除键盘事件监听
  window.removeEventListener('keydown', handleKeyDown)
  
  // 移除主题变化监听
  window.removeEventListener('theme-change', updateTheme)
})
</script>

<style scoped>
.terminals-container {
  height: 100%;
  width: 100%;
  position: relative;
}

.terminal-wrapper {
  height: 100%;
  width: 100%;
  position: relative;
}

/* 添加Tabs样式覆盖 */
:deep(.el-tabs__header) {
  margin-bottom: 0;
  background-color: var(--panel-bg);
  border-bottom: 1px solid var(--border-color);
}

:deep(.el-tabs__item) {
  color: var(--text-color);
  transition: all 0.3s;
}

:deep(.el-tabs__item.is-active) {
  color: var(--primary-color);
  font-weight: bold;
}

:deep(.el-tabs__nav-scroll) {
  overflow-x: auto;
  white-space: nowrap;
}

:deep(.el-tabs__nav) {
  height: 40px;
  line-height: 40px;
}
</style> 