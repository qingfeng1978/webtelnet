<!-- TerminalSearch.vue -->
<template>
  <div class="terminal-search-container" :class="{ 'dark-theme': isDarkTheme }">
    <div class="search-input-group">
      <el-input
        v-model="searchText"
        placeholder="在终端中搜索..."
        size="small"
        class="search-input"
        @keyup.enter="findNext"
        @keyup.esc="close"
        ref="searchInput"
      >
        <template #prefix>
          <el-icon><Search /></el-icon>
        </template>
      </el-input>
      
      <el-badge :value="resultCount > 0 ? resultCount : ''" :hidden="resultCount <= 0" type="info">
        <el-button-group>
          <el-button size="small" @click="findPrevious" :disabled="!canSearch">
            <el-icon><ArrowUp /></el-icon>
          </el-button>
          <el-button size="small" @click="findNext" :disabled="!canSearch">
            <el-icon><ArrowDown /></el-icon>
          </el-button>
        </el-button-group>
      </el-badge>
      
      <el-tooltip content="关闭搜索" placement="top">
        <el-button size="small" circle @click="close">
          <el-icon><Close /></el-icon>
        </el-button>
      </el-tooltip>
    </div>
    
    <div v-if="noResults && searchAttempted" class="no-results">
      未找到结果
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch, defineProps, defineEmits } from 'vue'
import { Search, ArrowUp, ArrowDown, Close } from '@element-plus/icons-vue'

const props = defineProps<{
  activeTerminalId: string;
  terminalRef: any;
  isDarkTheme?: boolean;
}>()

const emit = defineEmits(['close', 'search-result'])

const searchText = ref('')
const currentIndex = ref(-1)
const resultCount = ref(0)
const searchResults = ref<string[]>([])
const searchAttempted = ref(false)
const searchInput = ref<any>(null)

// 计算属性
const canSearch = computed(() => searchText.value.length >= 1)
const noResults = computed(() => searchAttempted.value && resultCount.value === 0)

// 搜索终端内容
const searchTerminalContent = () => {
  if (!canSearch.value || !props.terminalRef) return

  searchAttempted.value = true
  
  try {
    // 尝试获取终端内容
    // 我们需要通过传入的终端引用获取内容
    if (!props.terminalRef.serializeAddon) {
      console.error('终端的序列化插件不可用')
      resultCount.value = 0
      return
    }
    
    // 获取终端内容
    const content = props.terminalRef.serializeAddon.serialize() || ''
    
    // 搜索所有匹配项
    const searchRegex = new RegExp(searchText.value, 'gi')
    searchResults.value = []
    
    let match
    while ((match = searchRegex.exec(content)) !== null) {
      searchResults.value.push(match[0])
    }
    
    resultCount.value = searchResults.value.length
    
    // 重置当前索引
    if (resultCount.value > 0 && currentIndex.value === -1) {
      currentIndex.value = 0
    } else if (resultCount.value === 0) {
      currentIndex.value = -1
    }
    
    emit('search-result', {
      count: resultCount.value,
      current: currentIndex.value
    })
    
    return resultCount.value > 0
  } catch (error) {
    console.error('搜索终端内容失败:', error)
    resultCount.value = 0
    currentIndex.value = -1
    return false
  }
}

// 高亮当前搜索结果
const highlightCurrentResult = () => {
  if (currentIndex.value >= 0 && currentIndex.value < resultCount.value && props.terminalRef) {
    try {
      // 这里使用SearchAddon方法(如果可用)或滚动到可见位置
      if (props.terminalRef.searchAddon) {
        props.terminalRef.searchAddon.findNext(searchText.value)
      } else {
        // 备用方案: 通常只有内容可见的滚动
        console.log('使用备用滚动方法')
      }
      
      emit('search-result', {
        count: resultCount.value,
        current: currentIndex.value
      })
    } catch (error) {
      console.error('高亮搜索结果失败:', error)
    }
  }
}

// 查找下一个匹配项
const findNext = () => {
  if (!canSearch.value) return
  
  // 首次搜索或当搜索文本更改时重新搜索
  if (currentIndex.value === -1 || searchAttempted.value === false) {
    searchTerminalContent()
  }
  
  if (resultCount.value > 0) {
    // 移动到下一个结果
    currentIndex.value = (currentIndex.value + 1) % resultCount.value
    highlightCurrentResult()
  }
}

// 查找上一个匹配项
const findPrevious = () => {
  if (!canSearch.value) return
  
  // 首次搜索或当搜索文本更改时重新搜索
  if (currentIndex.value === -1 || searchAttempted.value === false) {
    searchTerminalContent()
  }
  
  if (resultCount.value > 0) {
    // 移动到上一个结果
    currentIndex.value = (currentIndex.value - 1 + resultCount.value) % resultCount.value
    highlightCurrentResult()
  }
}

// 关闭搜索
const close = () => {
  emit('close')
}

// 监听搜索文本变化
watch(searchText, () => {
  // 文本变化时重置搜索状态
  currentIndex.value = -1
  searchAttempted.value = false
  
  // 如果内容为空，清空结果
  if (!searchText.value) {
    resultCount.value = 0
    searchResults.value = []
  }
})

// 组件挂载时聚焦搜索输入框
onMounted(() => {
  if (searchInput.value) {
    searchInput.value.focus()
  }
})

// 暴露方法给父组件
defineExpose({
  focus: () => {
    if (searchInput.value) {
      searchInput.value.focus()
    }
  },
  clear: () => {
    searchText.value = ''
    currentIndex.value = -1
    resultCount.value = 0
    searchResults.value = []
    searchAttempted.value = false
  }
})
</script>

<style scoped>
.terminal-search-container {
  position: absolute;
  top: 10px;
  right: 10px;
  z-index: 1000;
  background-color: var(--bg-color);
  border: 1px solid var(--border-color);
  border-radius: 4px;
  box-shadow: 0 2px 12px 0 rgba(0, 0, 0, 0.1);
  display: flex;
  flex-direction: column;
  padding: 8px;
  min-width: 280px;
}

.search-input-group {
  display: flex;
  align-items: center;
  gap: 4px;
}

.search-input {
  flex: 1;
}

.no-results {
  color: #ff4949;
  margin-top: 4px;
  font-size: 12px;
  text-align: left;
  padding-left: 4px;
}

.dark-theme {
  background-color: var(--panel-bg);
  border-color: var(--border-color);
}
</style> 