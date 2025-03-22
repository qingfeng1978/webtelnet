<!-- Terminal.vue -->
<template>
  <div class="terminal-container" ref="terminalContainer">
    <div ref="terminal"></div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref, watch } from 'vue'
import { Terminal as XTerm } from 'xterm'
import { FitAddon } from 'xterm-addon-fit'
import { WebLinksAddon } from 'xterm-addon-web-links'
import { SerializeAddon } from 'xterm-addon-serialize'  // 导入序列化插件
import { SearchAddon } from 'xterm-addon-search'  // 导入搜索插件
import 'xterm/css/xterm.css'

// 定义登录状态类型
type LoginState = 'none' | 'username' | 'password'

const props = defineProps<{
  host: string
  port: number
  protocol?: string
  username?: string
  password?: string
  id?: string  // 添加ID属性用于识别终端
}>()

// 添加终端初始大小常量
const DEFAULT_COLS = 120  // 默认列数
const DEFAULT_ROWS = 36   // 默认行数
const MIN_COLS = 80       // 最小列数
const MIN_ROWS = 24       // 最小行数

// 生成唯一会话ID
const generateSessionID = () => {
  return `term_${Date.now()}_${Math.random().toString(36).slice(2, 8)}`
}

const sessionID = ref(props.id || generateSessionID())
const terminal = ref<XTerm | null>(null)
const terminalContainer = ref<HTMLElement | null>(null)
const ws = ref<WebSocket | null>(null)
const fitAddon = ref<FitAddon | null>(null)
const loginState = ref<LoginState>('none')
const buffer = ref('')
const isConnected = ref(false)
const isTerminalActive = ref(true)
const reconnectAttempts = ref(0)
const maxReconnectAttempts = 3
const isVisible = ref(true)
const isManuallyDisconnected = ref(false)

// 添加序列化插件实例
const serializeAddon = ref<SerializeAddon | null>(null)
const searchAddon = ref<SearchAddon | null>(null)

// 添加用于保存终端内容的变量
const terminalContent = ref<string>('')

// 检查终端实例是否有效 - 优化版本
const isValidTerminal = (term: any): term is XTerm => {
  if (!term) {
    console.warn('终端实例为空')
    return false
  }
  
  // 必需的方法列表
  const requiredMethods = ['dispose', 'write', 'refresh', 'clear', 'resize', 'open']
  // 必需的属性列表
  const requiredProps = ['element', 'cols', 'rows']
  
  try {
    // 检查是否为XTerm实例
    if (!(term instanceof XTerm)) {
      console.warn('对象不是XTerm实例')
      return false
    }
    
    // 批量检查必需的方法
    for (const method of requiredMethods) {
      if (typeof (term as Record<string, any>)[method] !== 'function') {
        console.warn(`终端缺少必需的方法: ${method}`)
        return false
      }
    }
    
    // 批量检查必需的属性
    for (const prop of requiredProps) {
      if (!(prop in term)) {
        console.warn(`终端缺少必需的属性: ${prop}`)
        return false
      }
    }
    
    // 检查特定属性类型
    if (typeof term.cols !== 'number' || typeof term.rows !== 'number') {
      console.warn('终端的cols或rows属性类型不正确')
      return false
    }
    
    return true
  } catch (error) {
    console.error('检查终端实例有效性时出错:', error)
    return false
  }
}

// 添加专门用于调整大小的方法
const resize = () => {
  console.log('手动调整终端大小...')
  if (terminal.value && isValidTerminal(terminal.value)) {
    try {
      // 先尝试设置为固定大小
      if (typeof terminal.value.resize === 'function') {
        terminal.value.resize(DEFAULT_COLS, DEFAULT_ROWS)
        console.log(`终端已重置为基础大小: ${DEFAULT_COLS}x${DEFAULT_ROWS}`)
      }
      
      // 然后通过fitAddon适应容器
      resizeTerminal()
      
      // 确保终端填满容器
      if (terminalContainer.value && terminal.value.element) {
        try {
          const termElement = terminal.value.element as HTMLElement;
          termElement.style.width = '100%';
          termElement.style.height = '100%';
          
          // 直接使用querySelector来获取单个canvas元素
          const canvas = terminal.value.element.querySelector('.xterm-screen canvas') as HTMLCanvasElement;
          if (canvas) {
            canvas.style.width = '100%';
            canvas.style.height = '100%';
          }
        } catch (styleError) {
          console.error('设置终端样式失败:', styleError);
        }
      }
      
      // 刷新显示
      if (typeof terminal.value.refresh === 'function') {
        terminal.value.refresh(0, terminal.value.rows - 1)
        console.log('终端显示已刷新')
      }
      
      return true
    } catch (error) {
      console.error('手动调整终端大小失败:', error)
      return false
    }
  } else {
    console.warn('终端不存在或无效，无法调整大小')
    return false
  }
}

// 定义终端活跃状态的getter和setter
const setIsTerminalActive = (value: boolean) => {
  // 由于现在我们要保持所有终端活跃，所以isTerminalActive始终为true
  // 我们使用isVisible来控制UI显示
  console.log(`设置终端UI可见性: ${value}`)
  
  // 如果值没有变化，不做处理
  if (isVisible.value === value) {
    console.log('可见性状态未变化，跳过处理')
    return
  }
  
  // 如果终端变为不可见，先保存当前终端内容
  if (!value && terminal.value && serializeAddon.value) {
    try {
      terminalContent.value = serializeAddon.value.serialize()
      console.log('终端内容已保存，长度:', terminalContent.value.length)
    } catch (error) {
      console.error('保存终端内容失败:', error)
    }
  }
  
  isVisible.value = value
  
  // 如果终端变为可见
  if (value) {
    console.log('终端变为可见，调整大小和刷新显示')
    
    // 确保终端存在或初始化
    if (!terminal.value || !isValidTerminal(terminal.value)) {
      console.log('终端实例无效或不存在，初始化终端')
      setTimeout(() => {
        const success = initTerminal()
        
        if (success) {
          // 重要：显式设置终端大小
          if (terminal.value && typeof terminal.value.resize === 'function') {
            terminal.value.resize(DEFAULT_COLS, DEFAULT_ROWS)
            console.log(`终端已设置为固定大小: ${DEFAULT_COLS}x${DEFAULT_ROWS}`)
          }
          
          setTimeout(() => {
            // 确保WebSocket连接
            if (!ws.value || ws.value.readyState === WebSocket.CLOSED) {
              console.log('WebSocket连接不可用，建立连接')
              connectWebSocket()
            } else if (ws.value.readyState === WebSocket.OPEN) {
              console.log('WebSocket连接正常')
              isConnected.value = true
            }
            
            // 调用resize确保终端尺寸正确
            resize()
          }, 100)
        }
      }, 50)
    } else {
      // 终端存在，只需调整大小
      console.log('终端实例有效，调整大小')
      try {
        // 检查WebSocket连接
        if (!ws.value || ws.value.readyState === WebSocket.CLOSED) {
          console.log('WebSocket连接断开，重新连接')
          connectWebSocket()
        } else if (ws.value.readyState === WebSocket.OPEN) {
          console.log('WebSocket连接正常')
          isConnected.value = true
        }
        
        // 重要：在调用resizeTerminal前确保终端有固定大小
        if (terminal.value && typeof terminal.value.resize === 'function' && 
            (!terminal.value.cols || !terminal.value.rows || 
             terminal.value.cols < MIN_COLS || terminal.value.rows < MIN_ROWS)) {
          terminal.value.resize(DEFAULT_COLS, DEFAULT_ROWS)
          console.log(`终端恢复到固定大小: ${DEFAULT_COLS}x${DEFAULT_ROWS}`)
        }
        
        // 调整终端大小并刷新显示
        resizeTerminal()
        
        // 额外刷新一次终端内容
        if (terminalContent.value && terminal.value && typeof terminal.value.refresh === 'function') {
          try {
            terminal.value.refresh(0, terminal.value.rows - 1)
            console.log('终端内容已刷新')
          } catch (refreshError) {
            console.error('刷新终端内容失败:', refreshError)
          }
        }
      } catch (error) {
        console.error('调整终端大小失败:', error)
      }
    }
  } else {
    // 变为不可见，但保持连接活跃
    console.log('终端变为不可见，但保持连接活跃')
  }
}

// 调整终端大小的辅助函数
const resizeTerminal = () => {
  if (terminal.value && fitAddon.value) {
    try {
      // 记录调整前的大小
      const oldCols = terminal.value.cols || DEFAULT_COLS
      const oldRows = terminal.value.rows || DEFAULT_ROWS
      
      // 尝试自动调整大小
      fitAddon.value.fit()
      
      // 检查调整后的大小是否合理
      if (!terminal.value.cols || !terminal.value.rows || 
          terminal.value.cols < MIN_COLS || terminal.value.rows < MIN_ROWS) {
        console.log('终端大小不合理，设置为默认大小')
        // 如果大小不合理，设置为默认大小或上次的大小
        terminal.value.resize(
          Math.max(oldCols, DEFAULT_COLS), 
          Math.max(oldRows, DEFAULT_ROWS)
        )
      }
      
      console.log(`终端大小已调整: ${terminal.value.cols}x${terminal.value.rows}`)
      
      // 确保终端元素使用100%尺寸
      if (terminal.value.element) {
        try {
          const termElement = terminal.value.element as HTMLElement;
          termElement.style.width = '100%';
          termElement.style.height = '100%';
          
          // 使用较安全的方式获取canvas元素
          const canvasElements = terminal.value.element.querySelectorAll('canvas');
          if (canvasElements && canvasElements.length > 0) {
            for (let i = 0; i < canvasElements.length; i++) {
              const canvasElement = canvasElements[i] as HTMLCanvasElement;
              canvasElement.style.width = '100%';
              canvasElement.style.height = '100%';
            }
          }
        } catch (styleError) {
          console.error('调整终端元素样式失败:', styleError);
        }
      }
      
      // 刷新终端显示
      if (typeof terminal.value.refresh === 'function') {
        terminal.value.refresh(0, terminal.value.rows - 1)
        console.log('终端显示已刷新')
      }
    } catch (error) {
      console.error('调整终端大小失败:', error)
      // 出错时设置为默认大小
      if (terminal.value && typeof terminal.value.resize === 'function') {
        try {
          terminal.value.resize(DEFAULT_COLS, DEFAULT_ROWS)
          console.log(`终端大小已重置为默认: ${DEFAULT_COLS}x${DEFAULT_ROWS}`)
        } catch (e) {
          console.error('重置终端大小失败:', e)
        }
      }
    }
  }
}

// 聚焦终端的方法
const focusTerminal = () => {
  if (terminal.value) {
    try {
      terminal.value.focus()
      console.log('终端已获得焦点')
    } catch (error) {
      console.error('聚焦终端失败:', error)
    }
  }
}

// 修改初始化终端函数，确保活跃状态
const initTerminal = () => {
  console.log('初始化终端...')
  
  // 确保现有终端被清理
  if (terminal.value) {
    try {
      // 在清理之前保存终端内容
      if (isValidTerminal(terminal.value) && terminal.value.element && serializeAddon.value) {
        try {
          console.log('保存终端内容状态')
          terminalContent.value = serializeAddon.value.serialize()
          console.log('终端内容已保存，长度:', terminalContent.value.length)
        } catch (serializeError) {
          console.error('保存终端内容失败:', serializeError)
        }
      }
      
      // 检查终端是否有效且有dispose方法
      if (isValidTerminal(terminal.value)) {
        console.log('清理现有终端实例')
        terminal.value.dispose()
      } else {
        console.log('终端实例无效，无法清理')
      }
    } catch (error) {
      console.error('清理旧终端失败:', error)
    }
    
    terminal.value = null
  }
  
  try {
    // 确保容器存在
    if (!terminalContainer.value) {
      throw new Error('终端容器不存在')
    }

    // 创建新的div元素作为终端容器
    const terminalElement = document.createElement('div')
    Object.assign(terminalElement.style, {
      width: '100%',
      height: '100%',
      position: 'absolute',
      top: '0',
      left: '0',
      right: '0',
      bottom: '0',
      backgroundColor: '#1e1e1e'
    })
    
    // 清空容器内容
    terminalContainer.value.innerHTML = ''
    terminalContainer.value.appendChild(terminalElement)

    // 创建新终端实例
    console.log('创建新终端实例')
    const terminalOptions = {
      cursorBlink: true,
      theme: {
        background: 'var(--terminal-bg)',
        foreground: 'var(--text-color)',
        cursor: 'var(--primary-color)',
        selection: 'rgba(64, 158, 255, 0.3)',
        black: '#000000',
        red: '#e06c75',
        green: '#98c379',
        yellow: '#e5c07b',
        blue: '#61afef',
        magenta: '#c678dd',
        cyan: '#56b6c2',
        white: '#dcdfe4',
        brightBlack: '#7f848e',
        brightRed: '#e06c75',
        brightGreen: '#98c379',
        brightYellow: '#e5c07b',
        brightBlue: '#61afef',
        brightMagenta: '#c678dd',
        brightCyan: '#56b6c2',
        brightWhite: '#ffffff'
      },
      fontSize: 16,
      fontFamily: '"Microsoft YaHei Mono", "Sarasa Mono SC", "更纱黑体 Mono SC", "Noto Sans Mono CJK SC", "Source Han Mono SC", "WenQuanYi Micro Hei Mono", Consolas, monospace',
      letterSpacing: 0,
      fontWeight: 'normal',
      fontWeightBold: 'bold',
      scrollback: 3000,
      allowTransparency: true,
      convertEol: true,
      cols: DEFAULT_COLS,
      rows: DEFAULT_ROWS,
      rendererType: 'canvas',
      disableStdin: false,
      rightClickSelectsWord: false,
      copyOnSelect: true,
      termProgram: 'vt100',
      screenKeys: true,
      cancelEvents: false,
      useFlowControl: true,
      tty: true,
      macOptionIsMeta: true,
      altClickMovesCursor: false,
      windowsMode: true,
      lineHeight: 1.2,
      charset: 'GB2312',
      encoding: 'gb2312'
    } as const

    console.log('终端配置:', JSON.stringify(terminalOptions, null, 2))

    try {
      // 确保XTerm构造函数可用
      if (typeof XTerm !== 'function') {
        throw new Error('XTerm构造函数未定义')
      }
      
      // 创建终端实例
      terminal.value = new XTerm(terminalOptions)
      console.log('终端实例创建成功，检查实例属性:', {
        hasTerminal: !!terminal.value,
        isXTermInstance: terminal.value instanceof XTerm,
        hasMethods: {
          dispose: typeof terminal.value?.dispose === 'function',
          write: typeof terminal.value?.write === 'function',
          refresh: typeof terminal.value?.refresh === 'function',
          clear: typeof terminal.value?.clear === 'function',
          resize: typeof terminal.value?.resize === 'function',
          open: typeof terminal.value?.open === 'function'
        }
      })
    } catch (error: any) {
      console.error('创建终端实例时发生错误:', error)
      throw new Error(`终端实例创建失败: ${error.message}`)
    }

    // 立即验证终端实例
    if (!terminal.value) {
      throw new Error('终端实例创建后为空')
    }

    console.log('验证终端实例...')
    if (!isValidTerminal(terminal.value)) {
      const terminalState = {
        type: typeof terminal.value,
        hasInstance: terminal.value ? ((terminal.value as any) instanceof XTerm) : false,
        methods: {
          dispose: typeof (terminal.value as any)?.dispose,
          write: typeof (terminal.value as any)?.write,
          refresh: typeof (terminal.value as any)?.refresh,
          clear: typeof (terminal.value as any)?.clear,
          resize: typeof (terminal.value as any)?.resize,
          open: typeof (terminal.value as any)?.open
        },
        properties: {
          hasElement: !!(terminal.value as any)?.element,
          colsType: typeof (terminal.value as any)?.cols,
          rowsType: typeof (terminal.value as any)?.rows
        }
      }
      console.error('终端实例验证失败，当前状态:', terminalState)
      throw new Error('创建的终端实例无效')
    }
    console.log('终端实例验证通过')

    // 设置状态
    isTerminalActive.value = true
    isConnected.value = false

    // 载入插件
    try {
      console.log('开始加载终端插件');
      const pluginsInitialized = initializeAddons();
      if (!pluginsInitialized) {
        throw new Error('插件初始化失败');
      }
      console.log('所有终端插件加载完成');
    } catch (addonError) {
      console.error('加载终端插件失败:', addonError);
      throw addonError;
    }

    // 打开终端
    try {
      console.log('在容器中打开终端')
      if (!terminalElement) {
        throw new Error('终端DOM元素不存在')
      }
      terminal.value.open(terminalElement)
      console.log('终端已成功打开')
      
      // 验证终端是否正确打开
      if (!terminal.value.element || !terminal.value.element.parentElement) {
        throw new Error('终端未正确附加到DOM')
      }
    } catch (openError) {
      console.error('打开终端失败:', openError)
      throw openError
    }

    // 恢复之前保存的终端内容
    if (terminalContent.value && terminalContent.value.length > 0) {
      try {
        console.log('正在恢复终端内容...')
        setTimeout(() => {
          if (terminal.value && typeof terminal.value.write === 'function') {
            const chunkSize = 5000
            const chunks = Math.ceil(terminalContent.value.length / chunkSize)
            
            console.log(`分${chunks}块恢复终端内容，总长度: ${terminalContent.value.length}字符`)
            
            for (let i = 0; i < chunks; i++) {
              const start = i * chunkSize
              const end = Math.min((i + 1) * chunkSize, terminalContent.value.length)
              const chunk = terminalContent.value.substring(start, end)
              
              setTimeout(() => {
                if (terminal.value && typeof terminal.value.write === 'function') {
                  try {
                    terminal.value.write(chunk)
                    console.log(`已恢复终端内容块 ${i+1}/${chunks}`)
                    
                    if (i === chunks - 1) {
                      setTimeout(() => {
                        if (terminal.value && typeof terminal.value.refresh === 'function') {
                          try {
                            terminal.value.refresh(0, terminal.value.rows - 1)
                            console.log('终端内容恢复完成，已刷新显示')
                          } catch (refreshError) {
                            console.error('刷新终端显示失败:', refreshError)
                          }
                        }
                      }, 50)
                    }
                  } catch (writeError) {
                    console.error(`写入终端内容块 ${i+1}/${chunks} 失败:`, writeError)
                  }
                }
              }, i * 30)
            }
          }
        }, 100)
      } catch (restoreError) {
        console.error('恢复终端内容失败:', restoreError)
      }
    }

    // 设置终端事件监听
    if (terminal.value) {
      terminal.value.onData((data) => {
        if (ws.value?.readyState === WebSocket.OPEN) {
          try {
            // 处理退格键
            if (data === '\x7f' || data === '\b') {
              // 对于telnet，发送标准的退格序列
              if (props.protocol === 'telnet') {
                ws.value.send('\x08\x7f')  // 发送BS (Backspace) 和 DEL
              } else {
                ws.value.send(data)  // 其他协议直接发送原始字符
              }
            } else {
              ws.value.send(data)
            }
          } catch (error) {
            console.error('发送终端输入数据失败:', error)
            if (error instanceof Error && error.message.includes('closed')) {
              console.log('WebSocket发送失败，连接似乎已关闭，尝试重连')
              setTimeout(connectWebSocket, 500)
            }
          }
        } else if (ws.value?.readyState === WebSocket.CONNECTING) {
          console.log('WebSocket正在连接中，输入被缓存')
        } else {
          console.log('WebSocket未连接，尝试重新连接')
          if (!ws.value || ws.value.readyState === WebSocket.CLOSED) {
            console.log('检测到WebSocket连接已关闭，尝试重新连接')
            setTimeout(connectWebSocket, 500)
          }
        }
      })

      // 添加选中文本事件监听
      terminal.value.onSelectionChange(() => {
        handleSelection()
      })

      // 添加DOM事件监听
      if (terminal.value.element) {
        terminal.value.element.addEventListener('contextmenu', handlePaste)
        terminal.value.element.addEventListener('focus', () => {
          console.log('终端获得焦点')
        })
        
        terminal.value.element.addEventListener('blur', () => {
          console.log('终端失去焦点')
        })
      }
    }

    // 延迟调整终端大小
    setTimeout(() => {
      if (terminal.value && isValidTerminal(terminal.value)) {
        try {
          console.log('调整终端大小以填满容器')
          terminal.value.resize(DEFAULT_COLS, DEFAULT_ROWS)
          fitAddon.value?.fit()
          console.log(`终端大小: ${terminal.value.cols}x${terminal.value.rows}`)
          
          if (typeof terminal.value.refresh === 'function') {
            terminal.value.refresh(0, terminal.value.rows - 1)
          }
          
          if (isVisible.value) {
            try {
              terminal.value.focus()
              console.log('终端已获得焦点')
            } catch (focusError) {
              console.error('终端获取焦点失败:', focusError)
            }
          }
        } catch (resizeError) {
          console.error('调整终端大小失败:', resizeError)
        }
      }
    }, 100)

    return true
  } catch (error) {
    console.error('初始化终端失败:', error)
    if (terminal.value) {
      try {
        terminal.value.dispose()
      } catch (disposeError) {
        console.error('清理失败的终端实例时出错:', disposeError)
      }
    }
    terminal.value = null
    return false
  }
}

const connectWebSocket = () => {
  console.log('连接WebSocket...')
  
  // 关闭现有连接
  if (ws.value) {
    try {
      if (ws.value.readyState === WebSocket.OPEN || ws.value.readyState === WebSocket.CONNECTING) {
        console.log('关闭现有WebSocket连接')
        ws.value.close()
      }
    } catch (error) {
      console.error('关闭WebSocket连接失败:', error)
    }
    ws.value = null
  }
  
  try {
    // 验证连接参数
    if (!props.host || !props.port) {
      console.error(`无效的连接参数: 主机=${props.host}, 端口=${props.port}`)
      if (terminal.value && isValidTerminal(terminal.value)) {
        terminal.value.write('\r\n错误: 无效的连接参数，请提供正确的主机和端口\r\n')
      }
      return
    }
    
    // 使用sessionID创建带有会话标识的WebSocket URL
    const wsUrl = `ws://localhost:8080/ws/${sessionID.value}`
    console.log(`创建新WebSocket连接: ${wsUrl} (sessionID: ${sessionID.value})`)
    
    // 详细记录连接参数
    console.log(`连接参数: 
      - 协议: ${props.protocol || 'telnet'}
      - 主机: ${props.host}
      - 端口: ${props.port}
      - 用户名: ${props.username ? '已提供' : '未提供'}
      - 密码: ${props.password ? '已提供' : '未提供'}
    `)
    
    ws.value = new WebSocket(wsUrl)
    
    // 设置连接超时检测
    const connectionTimeout = setTimeout(() => {
      if (ws.value && ws.value.readyState !== WebSocket.OPEN) {
        console.warn('WebSocket连接超时，关闭当前连接并重试')
        try {
          ws.value.close()
        } catch (e) {
          console.error('关闭超时连接失败:', e)
        }
        ws.value = null
        
        // 如果重试次数未超过限制，则重试
        if (reconnectAttempts.value < maxReconnectAttempts) {
          reconnectAttempts.value++
          const delay = Math.min(1000 * Math.pow(2, reconnectAttempts.value - 1), 5000)
          console.log(`WebSocket连接超时，${delay}ms后重试 (${reconnectAttempts.value}/${maxReconnectAttempts})`)
          setTimeout(connectWebSocket, delay)
        } else {
          console.error('WebSocket连接多次尝试失败，放弃连接')
          if (isVisible.value && terminal.value && typeof terminal.value.write === 'function') {
            try {
              terminal.value.write('\r\n连接多次尝试失败，请检查服务器状态或网络连接\r\n')
              terminal.value.write(`\r\n无法连接到 ${props.host}:${props.port}\r\n`)
              terminal.value.write('可能的原因:\r\n')
              terminal.value.write('1. 后端服务未运行\r\n')
              terminal.value.write('2. 网络连接问题\r\n')
              terminal.value.write('3. 目标服务器不可达\r\n')
              terminal.value.write('\r\n请检查并重试\r\n')
            } catch (writeError) {
              console.error('写入失败消息错误:', writeError)
            }
          }
        }
      }
    }, 8000) // 8秒超时

    ws.value.onopen = () => {
      console.log('WebSocket连接已打开')
      clearTimeout(connectionTimeout) // 清除连接超时
      isConnected.value = true
      reconnectAttempts.value = 0 // 重置重连次数
      
      // 准备连接信息
      const connectionInfo = {
        protocol: props.protocol || 'telnet',
        host: props.host,
        port: props.port,
        terminalType: 'vt100',
        encoding: 'GB2312',
        cols: terminal.value?.cols || DEFAULT_COLS,
        rows: terminal.value?.rows || DEFAULT_ROWS,
        ssh: props.protocol === 'ssh' ? {
          host: props.host,
          port: props.port,
          username: props.username || '',
          password: props.password || ''
        } : null
      }
      
      // 发送连接信息
      if (ws.value?.readyState === WebSocket.OPEN) {
        const logInfo = {
          ...connectionInfo,
          ssh: props.protocol === 'ssh' ? { 
            ...connectionInfo.ssh, 
            password: '******' 
          } : null
        };
        console.log(`发送连接信息: ${JSON.stringify(logInfo)}`)
        
        try {
          // 将连接信息序列化为JSON字符串并发送
          const jsonStr = JSON.stringify(connectionInfo)
          ws.value.send(jsonStr)
          console.log(`已发送连接信息 (${jsonStr.length} 字节)`)
        } catch (sendError) {
          console.error('发送连接信息失败:', sendError)
          if (isVisible.value && terminal.value && typeof terminal.value.write === 'function') {
            terminal.value.write('\r\n发送连接信息失败，请刷新页面重试\r\n')
          }
        }
      } else {
        console.error('WebSocket连接已关闭，无法发送连接信息')
      }
      
      // 检查终端是否有效并活跃
      if (isVisible.value && terminal.value && typeof terminal.value.write === 'function') {
        terminal.value.write(`正在连接到 ${props.host}:${props.port}...\r\n`)
      } else {
        console.warn('终端不可写，无法显示连接消息')
      }
    }

    ws.value.onmessage = (event) => {
      // 使用一个函数来安全地写入数据到终端
      const safeWriteToTerminal = (data: Uint8Array) => {
        if (!terminal.value) {
          console.log('终端不存在，无法写入数据')
          return false
        }
        
        // 检查终端是否有效
        if (!isValidTerminal(terminal.value)) {
          console.warn('终端已损坏，尝试重新初始化')
          
          // 尝试重新初始化终端
          initTerminal()
          return false
        }
        
        // 确保write方法存在
        if (typeof terminal.value.write !== 'function') {
          console.warn('终端.write不是函数，无法写入数据')
          return false
        }
        
        try {
          // 无论终端是否可见，都接收数据并更新内容
          terminal.value.write(data)
          
          // 如果终端内容发生变化，保存最新状态
          if (serializeAddon.value) {
            try {
              // 定期保存终端内容（这里可以优化为按需保存）
              terminalContent.value = serializeAddon.value.serialize()
            } catch (serializeError) {
              console.error('保存终端内容失败:', serializeError)
            }
          }
          
          return true
        } catch (error) {
          console.error('写入终端数据失败:', error)
          
          // 尝试恢复终端状态
          try {
            console.log('尝试恢复终端状态')
            setTimeout(() => {
              if (terminal.value && typeof terminal.value.refresh === 'function') {
                terminal.value.refresh(0, terminal.value.rows - 1)
              }
            }, 100)
          } catch (refreshError) {
            console.error('恢复终端状态失败:', refreshError)
          }
          
          return false
        }
      }
      
      try {
        // 高效处理不同类型的数据
        let dataToProcess: Uint8Array | null = null;
        
        // 处理Blob数据
        if (event.data instanceof Blob) {
          const reader = new FileReader()
          reader.onload = () => {
            try {
              const data = new Uint8Array(reader.result as ArrayBuffer)
              processData(data)
            } catch (error) {
              console.error('处理Blob数据失败:', error)
            }
          }
          reader.onerror = (fileErr) => {
            console.error('读取Blob数据失败:', fileErr)
          }
          reader.readAsArrayBuffer(event.data)
          return // 提前返回，让异步操作完成
        }
        // 处理ArrayBuffer数据
        else if (event.data instanceof ArrayBuffer) {
          dataToProcess = new Uint8Array(event.data)
        }
        // 处理字符串数据
        else if (typeof event.data === 'string') {
          // 使用TextEncoder编码字符串数据
          const encoder = new TextEncoder()
          dataToProcess = encoder.encode(event.data)
        }
        
        // 如果有有效数据，处理它
        if (dataToProcess) {
          processData(dataToProcess)
        }
        
        // 数据处理函数
        function processData(data: Uint8Array) {
          try {
            // 尝试使用GB2312解码
            const decoder = new TextDecoder('GB2312', { fatal: false })
            const text = decoder.decode(data)
            
            // 将解码后的文本写入终端
            safeWriteToTerminal(new TextEncoder().encode(text))
            
            // 处理缓冲区数据分析和自动登录
            const lines = (buffer.value + text).split('\n')
            buffer.value = lines.slice(-5).join('\n') // 只保留最后5行
            
            // 根据不同协议处理登录
            if (props.protocol === 'telnet' && props.username && props.password) {
              handleTelnetLogin(buffer.value)
            } else if (props.protocol === 'ssh' && props.username && props.password) {
              handleSSHLogin(buffer.value)
            }
          } catch (error) {
            console.error('处理数据失败:', error)
            // 如果GB2312解码失败，尝试使用其他编码方式
            try {
              // 尝试使用GBK解码
              const gbkDecoder = new TextDecoder('GBK', { fatal: false })
              const gbkText = gbkDecoder.decode(data)
              safeWriteToTerminal(new TextEncoder().encode(gbkText))
            } catch (gbkError) {
              console.error('GBK解码失败:', gbkError)
              // 如果所有解码方式都失败，直接写入原始数据
              safeWriteToTerminal(data)
            }
          }
        }
      } catch (error) {
        console.error('WebSocket消息处理错误:', error)
      }
    }

    ws.value.onerror = (event) => {
      console.error('WebSocket连接错误:', event)
      clearTimeout(connectionTimeout) // 清除连接超时
      
      if (isVisible.value && terminal.value && typeof terminal.value.write === 'function') {
        try {
          terminal.value.write('\r\n连接错误: WebSocket连接失败\r\n')
          terminal.value.write('请检查网络连接和后端服务状态\r\n')
        } catch (writeError) {
          console.error('写入错误信息失败:', writeError)
        }
      }
      
      // 如果连接尚未关闭，则关闭它
      try {
        if (ws.value && (ws.value.readyState === WebSocket.OPEN || ws.value.readyState === WebSocket.CONNECTING)) {
          ws.value.close()
        }
      } catch (closeError) {
        console.error('关闭WebSocket连接失败:', closeError)
      }
      
      // 如果是手动断开，不尝试重连
      if (isManuallyDisconnected.value) {
        console.log('连接由用户手动断开，不自动重连')
        return
      }
      
      // 错误重连逻辑
      if (reconnectAttempts.value < maxReconnectAttempts) {
        reconnectAttempts.value++
        const delay = Math.min(1000 * reconnectAttempts.value, 5000)
        console.log(`连接错误，${delay}ms后重试 (${reconnectAttempts.value}/${maxReconnectAttempts})`)
        setTimeout(connectWebSocket, delay)
      } else {
        console.error('多次尝试重连失败，放弃连接')
      }
    }

    ws.value.onclose = (event) => {
      console.log(`WebSocket连接已关闭: 代码=${event.code}, 原因=${event.reason || '未提供'}, 是否干净=${event.wasClean}`)
      clearTimeout(connectionTimeout) // 清除连接超时
      isConnected.value = false
      
      // 在终端显示连接关闭消息
      if (isVisible.value && terminal.value && typeof terminal.value.write === 'function') {
        try {
          // 如果不是手动断开，则显示断开连接的消息
          if (!isManuallyDisconnected.value) {
            terminal.value.write('\r\n\r\n连接已断开')
            
            // 显示重连信息
            if (reconnectAttempts.value < maxReconnectAttempts) {
              terminal.value.write('，正在尝试重新连接...\r\n')
            } else {
              terminal.value.write('\r\n请手动刷新页面重连\r\n')
            }
          }
        } catch (writeError) {
          console.error('写入断开连接消息失败:', writeError)
        }
      }
      
      // 如果是手动断开，不尝试重连
      if (isManuallyDisconnected.value) {
        console.log('连接由用户手动断开，不自动重连')
        isManuallyDisconnected.value = false // 重置状态
        return
      }
      
      // 自动重连逻辑
      if (reconnectAttempts.value < maxReconnectAttempts) {
        reconnectAttempts.value++
        const baseDelay = 1000 * reconnectAttempts.value
        const jitter = Math.floor(Math.random() * 1000) // 添加抖动，避免多个终端同时重连
        const delay = Math.min(baseDelay + jitter, 10000)
        console.log(`连接断开，${delay}ms后重试 (${reconnectAttempts.value}/${maxReconnectAttempts})`)
        
        setTimeout(connectWebSocket, delay)
      } else {
        console.error('多次尝试重连失败，放弃连接')
        
        // 显示最终失败消息
        if (isVisible.value && terminal.value && typeof terminal.value.write === 'function') {
          try {
            terminal.value.write('\r\n\r\n连接多次尝试失败，请尝试以下操作:\r\n')
            terminal.value.write('1. 检查网络连接\r\n')
            terminal.value.write('2. 确认服务器地址和端口是否正确\r\n')
            terminal.value.write('3. 重新启动后端服务\r\n')
            terminal.value.write('4. 刷新页面重试\r\n\r\n')
          } catch (writeError) {
            console.error('写入最终失败消息错误:', writeError)
          }
        }
      }
    }
  } catch (error) {
    console.error('创建WebSocket连接失败:', error)
    isConnected.value = false
    
    // 记录失败并显示在终端
    if (isVisible.value && terminal.value && typeof terminal.value.write === 'function') {
      try {
        terminal.value.write('\r\n创建WebSocket连接失败，请检查网络连接\r\n')
      } catch (writeError) {
        console.error('写入连接失败消息失败:', writeError)
      }
    }
    
    // 如果是因为创建失败，也尝试重连
    if (reconnectAttempts.value < maxReconnectAttempts) {
      reconnectAttempts.value = maxReconnectAttempts // 直接设置为最大值，防止重连
      console.log('连接创建失败，不进行重连')
      if (isVisible.value && terminal.value && typeof terminal.value.write === 'function') {
        try {
          terminal.value.write('\r\n连接失败，请手动重新连接\r\n')
        } catch (writeError) {
          console.error('写入连接失败消息失败:', writeError)
        }
      }
    }
  }
}

// 初始化插件
const initializeAddons = (): boolean => {
  console.log('开始加载终端插件...')
  
  if (!terminal.value || !isValidTerminal(terminal.value)) {
    console.error('终端实例无效，无法加载插件')
    return false
  }
  
  try {
    // 创建FitAddon实例
    console.log('加载FitAddon插件...')
    fitAddon.value = new FitAddon()
    terminal.value.loadAddon(fitAddon.value)
    console.log('FitAddon插件加载成功')
    
    // 创建WebLinksAddon实例
    console.log('加载WebLinksAddon插件...')
    const webLinksAddon = new WebLinksAddon()
    terminal.value.loadAddon(webLinksAddon)
    console.log('WebLinksAddon插件加载成功')
    
    // 创建SerializeAddon实例用于内容保存和恢复
    console.log('加载SerializeAddon插件...')
    serializeAddon.value = new SerializeAddon()
    terminal.value.loadAddon(serializeAddon.value)
    console.log('SerializeAddon插件加载成功')
    
    // 创建SearchAddon实例用于搜索功能
    console.log('加载SearchAddon插件...')
    searchAddon.value = new SearchAddon()
    terminal.value.loadAddon(searchAddon.value)
    console.log('SearchAddon插件加载成功')
    
    console.log('所有插件加载完成')
    return true
  } catch (error) {
    console.error('加载插件失败:', error)
    // 尝试清理已加载的插件
    try {
      if (fitAddon.value) {
        fitAddon.value.dispose()
        fitAddon.value = null
      }
      if (serializeAddon.value) {
        serializeAddon.value.dispose()
        serializeAddon.value = null
      }
      if (searchAddon.value) {
        searchAddon.value.dispose()
        searchAddon.value = null
      }
    } catch (disposeError) {
      console.error('清理插件失败:', disposeError)
    }
    
    return false
  }
}

// 安全地卸载插件
const disposeAddons = async () => {
  console.log('开始卸载终端插件...');
  
  const safeDisposeAddon = async (addon: any, name: string) => {
    if (!addon) {
      console.log(`${name} 不存在，跳过卸载`);
      return;
    }

    try {
      console.log(`正在卸载 ${name}...`);
      await addon.dispose();
      console.log(`${name} 卸载成功`);
    } catch (error) {
      console.warn(`${name} 卸载失败:`, error);
    }
  };

  await safeDisposeAddon(fitAddon.value, 'FitAddon');
  await safeDisposeAddon(serializeAddon.value, 'SerializeAddon');
  await safeDisposeAddon(searchAddon.value, 'SearchAddon');

  // 重置插件引用
  fitAddon.value = null;
  serializeAddon.value = null;
  searchAddon.value = null;
  console.log('插件引用已重置');
};

// 断开终端连接的函数
const disconnect = () => {
  console.log(`断开终端连接: ${sessionID.value}`)
  
  // 设置手动断开标志，避免自动重连
  isManuallyDisconnected.value = true

  // 关闭WebSocket连接
  if (ws.value) {
    try {
      // 在关闭前发送断开消息
      if (ws.value.readyState === WebSocket.OPEN) {
        try {
          const disconnectMsg = JSON.stringify({
            action: 'disconnect',
            sessionID: sessionID.value
          })
          ws.value.send(disconnectMsg)
          console.log('发送断开连接消息')
        } catch (sendError) {
          console.error('发送断开消息失败:', sendError)
        }
      }
      
      // 延迟一小段时间后关闭连接，确保消息发送
      setTimeout(() => {
        if (ws.value) {
          if (ws.value.readyState === WebSocket.OPEN || ws.value.readyState === WebSocket.CONNECTING) {
            console.log('关闭WebSocket连接')
            ws.value.close(1000, '用户手动断开连接')
          }
          ws.value = null
        }
      }, 100)
    } catch (error) {
      console.error('关闭WebSocket连接失败:', error)
      ws.value = null
    }
  }

  // 重置状态
  isConnected.value = false
  reconnectAttempts.value = 0
  loginState.value = 'none'

  // 保存终端内容
  if (terminal.value && serializeAddon.value) {
    try {
      terminalContent.value = serializeAddon.value.serialize()
      console.log('终端内容已保存，长度:', terminalContent.value.length)
    } catch (error) {
      console.error('保存终端内容失败:', error)
    }
  }

  // 显示断开消息
  if (terminal.value && isValidTerminal(terminal.value)) {
    try {
      terminal.value.write('\r\n\r\n已断开连接\r\n')
    } catch (error) {
      console.error('写入断开消息失败:', error)
    }
  }

  // 清理缓冲区
  buffer.value = ''
  
  console.log('终端连接已断开')
  return true
}

// 监听属性变化
watch(() => [props.host, props.port, props.protocol], () => {
  console.log('属性变化，重新初始化终端')
  disconnect()
  
  // 短暂延迟后重新初始化
  setTimeout(() => {
    if (!isTerminalActive.value) {
      isTerminalActive.value = true
      initTerminal()
      connectWebSocket()
    }
  }, 100)
}, { deep: true })

onMounted(() => {
  console.log(`Terminal组件已挂载, sessionID: ${sessionID.value}`)
  
  // 初始化终端和连接，确保一直保持活跃
  const success = initTerminal()
  
  // 延迟确保终端初始化完成后再建立连接
  if (success) {
    setTimeout(() => {
      try {
        connectWebSocket()
      } catch (error) {
        console.error('初始连接WebSocket失败:', error)
      }
    }, 100)
  } else {
    console.warn('终端初始化失败，跳过WebSocket连接')
  }

  // 添加窗口大小调整事件
  const handleResize = () => {
    // 只调整可见终端的大小
    if (terminal.value && fitAddon && isVisible.value) {
      try {
        fitAddon.value?.fit()
        console.log(`终端大小已调整: ${terminal.value.cols}x${terminal.value.rows}`)
        
        // 刷新终端显示
        if (terminal.value && typeof terminal.value.refresh === 'function') {
          terminal.value.refresh(0, terminal.value.rows - 1)
        }
      } catch (error) {
        console.error('调整终端大小失败:', error)
      }
    }
  }
  window.addEventListener('resize', handleResize)

  // 添加状态检查任务
  const statusCheckInterval = setInterval(() => {
    try {
      // 检查终端是否有效，无论是否可见都保持有效
      if (isVisible.value && (!terminal.value || !isValidTerminal(terminal.value))) {
        // 只有在终端可见且实例无效时才尝试重新初始化
        console.warn('状态检查: 终端实例无效，尝试重新初始化')
        initTerminal()
      } else if (!isVisible.value && (!terminal.value || !isValidTerminal(terminal.value))) {
        // 如果终端不可见但实例无效，只记录日志但不初始化
        console.log('状态检查: 终端不可见且实例无效，待切换回标签时再初始化')
      }
      
      // 检查WebSocket连接，无论是否可见都保持连接
      if (!ws.value || ws.value.readyState === WebSocket.CLOSED) {
        console.warn('状态检查: WebSocket连接断开，尝试重连')
        connectWebSocket()
      } else if (ws.value?.readyState === WebSocket.OPEN) {
        // 发送心跳
        try {
          ws.value.send(new Uint8Array([0]))
        } catch (error) {
          console.error('发送心跳失败:', error)
        }
      }
    } catch (error) {
      console.error('状态检查失败:', error)
    }
  }, 30000)
  
  // 添加定期保存终端内容的任务
  const contentSaveInterval = setInterval(() => {
    if (terminal.value && isValidTerminal(terminal.value) && serializeAddon.value) {
      try {
        const oldContentLength = terminalContent.value.length
        terminalContent.value = serializeAddon.value.serialize()
        const newContentLength = terminalContent.value.length
        
        if (newContentLength !== oldContentLength) {
          console.log(`定期保存终端内容：${oldContentLength} -> ${newContentLength} 字节`)
        }
      } catch (error) {
        console.error('定期保存终端内容失败:', error)
      }
    }
  }, 5000)

  // 清理资源
  onUnmounted(() => {
    console.log('Terminal组件将卸载')
    window.removeEventListener('resize', handleResize)
    clearInterval(statusCheckInterval)
    clearInterval(contentSaveInterval)
    disconnect()
    
    // 清理事件监听
    if (terminal.value?.element) {
      terminal.value.element.removeEventListener('contextmenu', handlePaste)
    }
  })
})

// 更新公开的方法
defineExpose({
  disconnect,
  get isTerminalActive() { return true }, // 始终返回true保持活跃
  set isTerminalActive(value: boolean) { setIsTerminalActive(value) }, // 仅控制UI可见性
  focusTerminal, // 添加聚焦方法
  sessionID: sessionID.value, // 暴露会话ID供父组件使用
  resize, // 暴露resize方法给父组件
  serializeAddon, // 暴露序列化插件给父组件
  searchAddon // 暴露搜索插件给父组件
})

// 处理Telnet登录
function handleTelnetLogin(bufferText: string) {
  // 如果已经检测到命令提示符，说明已经登录成功，不再处理登录
  const commandPromptRegex = /[$#>]\s*$/
  const lastLine = bufferText.split('\n').pop()?.trim() || ''
  
  if (commandPromptRegex.test(lastLine)) {
    console.log('telnet已登录，检测到命令提示符')
    loginState.value = 'none' // 重置状态，防止重复登录
    buffer.value = '' // 清空缓冲区，避免之前的内容触发登录
    return
  }

  // Telnet登录逻辑（用户名、密码提示可能因服务器不同而不同）
  const usernamePrompts = ['login:', 'username:', 'user name:', 'user:']
  const passwordPrompts = ['password:', 'passwd:']
  
  // 检查是否有用户名提示
  if (usernamePrompts.some(prompt => lastLine.toLowerCase().includes(prompt))) {
    console.log('检测到telnet用户名提示')
    loginState.value = 'username'
    
    // 延迟发送用户名
    setTimeout(() => {
      if (ws.value?.readyState === WebSocket.OPEN && props.username) {
        console.log('发送telnet用户名:', props.username)
        ws.value?.send(props.username + '\r')
      }
    }, 500)
  } 
  // 检查是否有密码提示
  else if (passwordPrompts.some(prompt => lastLine.toLowerCase().includes(prompt))) {
    console.log('检测到telnet密码提示')
    loginState.value = 'password'
    
    // 延迟发送密码
    setTimeout(() => {
      if (ws.value?.readyState === WebSocket.OPEN && props.password) {
        console.log('发送telnet密码')
        ws.value?.send(props.password + '\r')
      }
    }, 500)
  }
}

// 处理SSH登录
function handleSSHLogin(bufferText: string) {
  // SSH登录通常会提示密码或密钥验证
  if (loginState.value === 'none') {
    // 检查密码提示
    if (bufferText.includes('password:')) {
      console.log('检测到SSH密码提示')
      loginState.value = 'password'
      
      // 延迟发送密码
      setTimeout(() => {
        if (ws.value?.readyState === WebSocket.OPEN) {
          console.log('发送SSH密码')
          ws.value?.send(props.password + '\r')
        }
      }, 100)
    }
  }
  
  // 检测登录成功（通常是出现命令提示符）
  if (loginState.value === 'password') {
    // 典型的命令提示符，例如$、#、>
    if (/[$#>]\s*$/.test(bufferText.trim())) {
      console.log('SSH登录成功，检测到命令提示符')
      loginState.value = 'none' // 重置状态，防止重复登录
    }
  }
}

// 添加剪贴板处理函数
const handlePaste = async (event: MouseEvent) => {
  event.preventDefault() // 阻止默认右键菜单
  
  try {
    const text = await navigator.clipboard.readText()
    if (ws.value?.readyState === WebSocket.OPEN && text) {
      ws.value.send(text)
    }
  } catch (error) {
    console.error('粘贴失败:', error)
  }
}

// 添加自动复制选中文本功能
const handleSelection = () => {
  if (terminal.value && terminal.value.hasSelection()) {
    const selectedText = terminal.value.getSelection()
    if (selectedText) {
      try {
        navigator.clipboard.writeText(selectedText)
        console.log('文本已复制到剪贴板')
      } catch (error) {
        console.error('复制到剪贴板失败:', error)
      }
    }
  }
}
</script>

<style scoped>
.terminal-container {
  width: 100%;
  height: 100%;
  min-height: 800px;
  min-width: 800px;
  background-color: #1e1e1e;
  padding: 0;
  margin: 0;
  text-align: left;
  display: flex;
  justify-content: flex-start;
  align-items: flex-start;
  overflow: hidden;
  box-sizing: border-box;
  position: relative;
  font-smooth: always;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
}

.terminal-container div {
  margin: 0;
  padding: 0;
  width: 100%;
  height: 100%;
  text-align: left;
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
}

/* xterm.js特定样式覆盖 */
:deep(.xterm) {
  padding: 0;
  margin: 0;
  text-align: left;
  height: 100% !important;
  width: 100% !important;
}

:deep(.xterm-viewport) {
  margin: 0;
  padding: 0;
  width: 100% !important;
  height: 100% !important;
}

:deep(.xterm-screen) {
  text-align: left;
  margin: 0;
  padding: 0;
  width: 100% !important;
  height: 100% !important;
}

:deep(.xterm-rows) {
  text-align: left;
  margin: 0;
  padding: 0;
  width: 100%;
}

/* 自定义滚动条样式 */
:deep(::-webkit-scrollbar) {
  width: 8px;
  height: 8px;
}

:deep(::-webkit-scrollbar-track) {
  background: transparent;
}

:deep(::-webkit-scrollbar-thumb) {
  background: #555;
  border-radius: 4px;
}

:deep(::-webkit-scrollbar-thumb:hover) {
  background: #666;
}

/* 添加更多canvas样式 */
:deep(.xterm-screen canvas) {
  width: 100% !important;
  height: 100% !important;
  object-fit: contain;
}

/* 确保字体渲染正确 */
:deep(.xterm .xterm-helpers) {
  position: absolute;
  top: 0;
  z-index: 5;
}

/* 确保终端容器使用全部可用空间 */
:deep(.terminal.xterm) {
  width: 100% !important;
  height: 100% !important;
  padding: 0 !important;
}

/* 确保canvas容器使用全部可用空间 */
:deep(.xterm-screen) {
  display: block;
  position: relative;
  width: 100% !important;
  height: auto !important;
  min-height: 100% !important;
}
</style> 

