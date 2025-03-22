<template>
  <div class="server-tree-container">
    <!-- 工具栏 -->
    <div class="toolbar">
      <!-- 删除这两个按钮 -->
      <!--
      <el-tooltip content="添加服务器" placement="bottom" :effect="isDarkTheme ? 'dark' : 'light'">
        <el-button type="primary" class="toolbar-btn" @click="handleAddServer">
          <el-icon><Plus /></el-icon>
        </el-button>
      </el-tooltip>
      
      <el-tooltip content="删除服务器" placement="bottom" :effect="isDarkTheme ? 'dark' : 'light'">
        <el-button 
          type="danger" 
          class="toolbar-btn" 
          @click="handleDeleteServer" 
          :disabled="!selectedServer || !selectedServer.id"
        >
          <el-icon><Delete /></el-icon>
        </el-button>
      </el-tooltip>
      -->

      <el-tooltip content="搜索服务器" placement="bottom" :effect="isDarkTheme ? 'dark' : 'light'">
        <el-input
          v-model="searchQuery"
          placeholder="搜索服务器"
          class="search-input"
          clearable
          @input="handleSearch"
        >
          <template #prefix>
            <el-icon><Search /></el-icon>
          </template>
        </el-input>
      </el-tooltip>
    </div>

    <!-- 树形控件容器 -->
    <div class="tree-container" @contextmenu.prevent="handleContainerRightClick">
      <div v-if="!serverTree.length" class="empty-state">
        <el-empty description="暂无服务器">
          <el-button type="primary" @click="handleAddServer">添加服务器</el-button>
        </el-empty>
      </div>
      <div v-else class="tree-content" :class="{'is-filtering': searchQuery}">
        <el-tree
          ref="treeRef"
          :data="serverTree"
          :props="defaultProps"
          :highlight-current="true"
          node-key="id"
          :expand-on-click-node="false"
          @node-click="handleNodeClick"
          @node-contextmenu="handleRightClick"
          :filter-node-method="filterNode"
        >
          <template #default="{ node, data }">
            <span 
              class="custom-tree-node" 
              :data-is-server="!!data.server.host"
              :data-is-folder="!data.server.host"
              :title="data.server.host 
                ? `双击连接到 ${data.server.name} (${data.server.host}:${data.server.port})` 
                : `双击展开/折叠 ${data.server.name}`"
              @dblclick.stop="handleNodeDblClick($event, data)"
            >
              <span class="node-icon">
                {{ !data.server.host ? '📁' : '🖥️' }}
              </span>
              <span class="node-label">{{ node.label }}</span>
            </span>
          </template>
        </el-tree>
      </div>
    </div>

    <!-- 右键菜单 -->
    <div v-show="showContextMenu" :style="contextMenuStyle" class="context-menu">
      <!-- 空白区域的右键菜单 -->
      <template v-if="contextMenuType === 'container'">
        <div class="context-menu-item" @click="handleAddRootFolder">
          <el-icon><Folder /></el-icon>
          <span>添加文件夹</span>
        </div>
        <div class="context-menu-item" @click="handleAddRootServer">
          <el-icon><Plus /></el-icon>
          <span>添加服务器</span>
        </div>
      </template>
      
      <!-- 文件夹节点的右键菜单 -->
      <template v-else-if="!contextMenuNode?.server.host">
        <div class="context-menu-item" @click="handleAddServer">
          <el-icon><Plus /></el-icon>
          <span>添加服务器</span>
        </div>
        <div class="context-menu-item" @click="handleAddFolder">
          <el-icon><Folder /></el-icon>
          <span>添加文件夹</span>
        </div>
        <div class="context-menu-item" @click="handleEdit">
          <el-icon><Edit /></el-icon>
          <span>编辑文件夹</span>
        </div>
        <div class="context-menu-item" @click="handleDeleteServer">
          <el-icon><Delete /></el-icon>
          <span>删除文件夹</span>
        </div>
      </template>
      
      <!-- 服务器节点的右键菜单 -->
      <template v-else>
        <div class="context-menu-item" @click="handleConnect">
          <el-icon><Connection /></el-icon>
          <span>连接</span>
        </div>
        <div class="context-menu-item" @click="handleEdit">
          <el-icon><Edit /></el-icon>
          <span>编辑</span>
        </div>
        <div class="context-menu-item" @click="handleMove">
          <el-icon><Position /></el-icon>
          <span>移动到</span>
        </div>
        <div class="context-menu-item" @click="handleDeleteServer">
          <el-icon><Delete /></el-icon>
          <span>删除</span>
        </div>
      </template>
    </div>

    <!-- 添加服务器对话框 -->
    <el-dialog
      v-model="showAddDialog"
      title="添加服务器"
      width="30%"
    >
      <el-form :model="newServer" label-width="80px">
        <el-form-item label="名称">
          <el-input v-model="newServer.name" placeholder="请输入名称"></el-input>
        </el-form-item>
        <el-form-item label="类型">
          <el-radio-group v-model="newServer.type">
            <el-radio :label="'folder'">文件夹</el-radio>
            <el-radio :label="'server'">服务器</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="上级" v-if="folders.length > 0">
          <el-select v-model="newServer.parentId" placeholder="选择上级">
            <el-option :value="0" label="根目录"></el-option>
            <el-option 
              v-for="folder in folders" 
              :key="folder.server.id" 
              :value="folder.server.id" 
              :label="folder.server.name">
            </el-option>
          </el-select>
        </el-form-item>
        <template v-if="newServer.type === 'server'">
          <el-form-item label="主机">
            <el-input v-model="newServer.host" placeholder="请输入主机地址"></el-input>
          </el-form-item>
          <el-form-item label="端口">
            <el-input-number v-model="newServer.port" :min="1" :max="65535"></el-input-number>
          </el-form-item>
          <el-form-item label="用户名">
            <el-input v-model="newServer.username" placeholder="可选"></el-input>
          </el-form-item>
          <el-form-item label="密码">
            <el-input v-model="newServer.password" type="password" placeholder="可选"></el-input>
          </el-form-item>
        </template>
      </el-form>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="showAddDialog = false">取消</el-button>
          <el-button type="primary" @click="createServer">确定</el-button>
        </span>
      </template>
    </el-dialog>

    <!-- 添加编辑服务器/文件夹对话框 -->
    <el-dialog
      v-model="showEditDialog"
      :title="editingServer.type === 'folder' ? '编辑文件夹' : '编辑服务器'"
      width="30%"
    >
      <el-form :model="editingServer" label-width="80px">
        <el-form-item label="名称">
          <el-input v-model="editingServer.name" placeholder="请输入名称"></el-input>
        </el-form-item>
        <template v-if="editingServer.type === 'server'">
          <el-form-item label="主机">
            <el-input v-model="editingServer.host" placeholder="请输入主机地址"></el-input>
          </el-form-item>
          <el-form-item label="端口">
            <el-input-number v-model="editingServer.port" :min="1" :max="65535"></el-input-number>
          </el-form-item>
          <el-form-item label="用户名">
            <el-input v-model="editingServer.username" placeholder="可选"></el-input>
          </el-form-item>
          <el-form-item label="密码">
            <el-input v-model="editingServer.password" type="password" placeholder="可选"></el-input>
          </el-form-item>
        </template>
      </el-form>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="showEditDialog = false">取消</el-button>
          <el-button type="primary" @click="updateServer">确定</el-button>
        </span>
      </template>
    </el-dialog>

    <!-- 移动服务器对话框 -->
    <el-dialog
      v-model="showMoveDialog"
      title="移动服务器"
      width="30%"
      :close-on-click-modal="false"
      :destroy-on-close="true"
    >
      <div v-if="moveTarget.id" class="move-dialog-content">
        <div class="move-source-info">
          <div class="source-label">移动对象:</div>
          <div class="source-value">
            {{ contextMenuNode?.server.name }}
            <span class="source-type">{{ !contextMenuNode?.server.host ? '(文件夹)' : '(服务器)' }}</span>
          </div>
        </div>
        
        <el-form :model="moveTarget" label-width="80px" class="move-form">
          <el-form-item label="目标位置">
            <el-select 
              v-model="moveTarget.parentId" 
              placeholder="选择目标文件夹" 
              class="full-width-select"
              :disabled="availableFolders.length === 0"
            >
              <el-option :value="0" label="根目录"></el-option>
              <el-option 
                v-for="folder in availableFolders" 
                :key="folder.server.id" 
                :value="folder.server.id" 
                :label="folder.server.name">
              </el-option>
            </el-select>
          </el-form-item>
        </el-form>
        
        <div v-if="availableFolders.length === 0" class="no-target-warning">
          <el-alert
            title="没有可用的目标文件夹"
            type="warning"
            :closable="false"
            show-icon
          >
            <p>当前无法移动此项目，因为没有可用的目标文件夹。</p>
          </el-alert>
        </div>
        
        <div v-if="isSameParent" class="move-warning">
          <el-alert
            title="目标位置未更改"
            type="info"
            :closable="false"
            show-icon
          >
            <p>您选择的目标位置与当前位置相同。</p>
          </el-alert>
        </div>
      </div>
      
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="cancelMove">取消</el-button>
          <el-button 
            type="primary" 
            @click="executeMove" 
            :disabled="isSameParent || availableFolders.length === 0"
            :loading="isMoving"
          >
            确定
          </el-button>
        </span>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed, onUnmounted, nextTick } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Delete, Search, Folder, Connection, Edit, Position } from '@element-plus/icons-vue'
import type { TreeInstance } from 'element-plus'

interface Server {
  id: number
  parent_id: number
  name: string
  host: string
  port: number
  username?: string
  password?: string
}

interface ServerTreeNode {
  server: Server
  children: ServerTreeNode[]
  type: 'folder' | 'server'
}

const emit = defineEmits(['select-server'])

const serverTree = ref<ServerTreeNode[]>([])
const selectedServer = ref<Server | null>(null)
const showAddDialog = ref(false)
const newServer = ref({
  name: '',
  type: 'server',
  parentId: 0,
  host: '',
  port: 23,
  username: '',
  password: ''
})

const defaultProps = {
  children: 'children',
  label: (data: ServerTreeNode) => data.server.name
}

// 计算所有文件夹
const folders = computed(() => {
  const result: ServerTreeNode[] = []
  const findFolders = (nodes: ServerTreeNode[]) => {
    for (const node of nodes) {
      if (!node.server.host) {
        result.push(node)
      }
      if (node.children && node.children.length > 0) {
        findFolders(node.children)
      }
    }
  }
  findFolders(serverTree.value)
  return result
})

// 添加主题状态变量
const isDarkTheme = ref(localStorage.getItem('theme') === 'dark')

// 监听主题变化
const updateTheme = () => {
  isDarkTheme.value = localStorage.getItem('theme') === 'dark'
  console.log('主题已更新:', isDarkTheme.value ? '深色' : '浅色')
}

// 处理自定义主题变化事件
const handleThemeChange = (event: CustomEvent) => {
  isDarkTheme.value = event.detail.theme === 'dark'
  console.log('接收到主题变化:', isDarkTheme.value ? '深色' : '浅色')
}

// 获取服务器列表
const fetchServers = async () => {
  try {
    const response = await fetch('http://localhost:8080/servers/tree')
    const data = await response.json()
    serverTree.value = data
  } catch (error) {
    console.error('Failed to fetch servers:', error)
    ElMessage.error('获取服务器列表失败')
  }
}

// 创建服务器
const createServer = async () => {
  try {
    const serverData = {
      parent_id: newServer.value.parentId,
      name: newServer.value.name,
      host: newServer.value.type === 'server' ? newServer.value.host : '',
      port: newServer.value.type === 'server' ? newServer.value.port : 0,
      username: newServer.value.username,
      password: newServer.value.password,
    }
    
    const response = await fetch('http://localhost:8080/servers', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(serverData)
    })
    
    if (response.ok) {
      ElMessage.success('添加成功')
      showAddDialog.value = false
      // 重置表单
      newServer.value = {
        name: '',
        type: 'server',
        parentId: 0,
        host: '',
        port: 23,
        username: '',
        password: ''
      }
      // 重新获取服务器列表
      await fetchServers()
    } else {
      ElMessage.error('添加失败')
    }
  } catch (error) {
    console.error('Failed to create server:', error)
    ElMessage.error('添加服务器失败')
  }

}

// 编辑对话框状态
const showEditDialog = ref(false)
const editingServer = ref({
  id: 0,
  name: '',
  type: 'server',
  host: '',
  port: 23,
  username: '',
  password: '',
  parent_id: 0
})

// 移动对话框状态
const showMoveDialog = ref(false)
const moveTarget = ref({
  id: 0,
  parentId: 0,
  originalParentId: 0 // 添加原始父ID用于比较
})
const isMoving = ref(false) // 添加移动中状态标记

// 树形组件引用
const treeRef = ref<TreeInstance>()

// 状态变量
const searchQuery = ref('')
const showContextMenu = ref(false)
const contextMenuNode = ref<ServerTreeNode | null>(null)
const contextMenuPosition = ref({ x: 0, y: 0 })
const contextMenuType = ref<'node' | 'container'>('node') // 右键菜单类型：节点或容器

// 计算属性
const contextMenuStyle = computed(() => ({
  left: `${contextMenuPosition.value.x}px`,
  top: `${contextMenuPosition.value.y}px`
}))

const isSameParent = computed(() => {
  return moveTarget.value.parentId === moveTarget.value.originalParentId;
})

// 方法定义
const handleSearch = (val: string) => {
  searchQuery.value = val
  treeRef.value?.filter(val)
}

const filterNode = (value: string, data: ServerTreeNode) => {
  if (!value) return true
  return data.server.name.toLowerCase().includes(value.toLowerCase())
}

const handleNodeClick = (node: ServerTreeNode) => {
  selectedServer.value = node.server
  // 仅选中，不连接
  console.log('选中服务器:', node.server)
}

// 修改handleNodeDblClick函数
const handleNodeDblClick = (event: Event, node: ServerTreeNode) => {
  // 添加调试信息
  console.log('双击事件触发', event, node);
  
  // 阻止事件冒泡，防止触发其他点击事件
  event.stopPropagation();
  
  // 如果双击的是服务器(不是文件夹)，触发连接
  if (node.server && node.server.host) {
    console.log('双击连接到服务器:', node.server);
    
    // 添加动画效果
    const target = event.currentTarget as HTMLElement;
    if (target) {
      target.classList.add('node-pulse');
      setTimeout(() => {
        target.classList.remove('node-pulse');
      }, 300);
    }
    
    // 调用父组件的连接方法
    try {
      console.log('发出select-server事件，数据:', node.server);
      emit('select-server', node.server);
      console.log('select-server事件已发出');
    } catch (error) {
      console.error('发出select-server事件失败:', error);
    }
    
    // 添加成功提示
    ElMessage.success(`正在连接到 ${node.server.name}`);
  } 
  // 如果双击的是文件夹，则展开/折叠节点
  else {
    console.log('双击展开/折叠文件夹:', node.server.name);
    
    // 添加动画效果
    const target = event.currentTarget as HTMLElement;
    if (target) {
      target.classList.add('folder-pulse');
      setTimeout(() => {
        target.classList.remove('folder-pulse');
      }, 300);
    }
    
    // 查找当前节点的展开/折叠图标并模拟点击
    // 获取节点容器
    const treeNodeEl = (event.currentTarget as HTMLElement)?.closest('.el-tree-node');
    if (treeNodeEl) {
      // 查找展开/折叠图标按钮
      const expandBtn = treeNodeEl.querySelector('.el-tree-node__expand-icon');
      if (expandBtn) {
        // 模拟点击展开/折叠按钮
        (expandBtn as HTMLElement).click();
      }
    }
  }
}

// 处理容器右键点击事件
const handleContainerRightClick = (event: MouseEvent) => {
  // 确保事件发生在空白区域，不是树节点
  if ((event.target as HTMLElement).closest('.el-tree-node')) {
    return; // 如果点击的是树节点，不处理
  }
  
  event.preventDefault();
  contextMenuPosition.value = { x: event.clientX, y: event.clientY };
  contextMenuNode.value = null; // 清空节点引用
  contextMenuType.value = 'container'; // 设置右键菜单类型为容器
  showContextMenu.value = true;
}

// 处理节点右键点击事件
const handleRightClick = (event: MouseEvent, node: ServerTreeNode) => {
  event.preventDefault();
  contextMenuPosition.value = { x: event.clientX, y: event.clientY };
  contextMenuNode.value = node;
  contextMenuType.value = 'node'; // 设置右键菜单类型为节点
  showContextMenu.value = true;
}

const handleAddServer = () => {
  showAddDialog.value = true
  newServer.value = {
    name: '',
    type: 'server',
    parentId: contextMenuNode.value?.server.id || 0,
    host: '',
    port: 23,
    username: '',
    password: ''
  }
}

const handleDeleteServer = async () => {
  if (!contextMenuNode.value) return;
  
  try {
    const isFolder = !contextMenuNode.value.server.host;
    const serverName = contextMenuNode.value.server.name;
    
    await ElMessageBox.confirm(
      `确定要删除${isFolder ? '文件夹' : '服务器'} "${serverName}" 吗？${isFolder ? '文件夹内的所有服务器也将被删除！' : ''}`, 
      '警告', 
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
        confirmButtonClass: 'el-button--danger'
      }
    )
    
    const response = await fetch(`http://localhost:8080/servers/${contextMenuNode.value.server.id}`, {
      method: 'DELETE'
    })
    
    if (response.ok) {
      ElMessage.success('删除成功')
      // 清空选中的服务器（如果被删除的就是当前选中的）
      if (selectedServer.value && selectedServer.value.id === contextMenuNode.value.server.id) {
        selectedServer.value = null
      }
      // 重新获取服务器列表
      await fetchServers()
    } else {
      const errorText = await response.text()
      console.error('删除失败:', errorText)
      ElMessage.error(`删除失败: ${response.status} ${response.statusText}`)
    }
  } catch (error) {
    if (error !== 'cancel') {
      console.error('Failed to delete server:', error)
      ElMessage.error('删除失败，请检查网络连接或服务器状态')
    }
  }
  
  showContextMenu.value = false
}

const handleAddFolder = () => {
  showAddDialog.value = true
  newServer.value = {
    name: '',
    type: 'folder',
    parentId: contextMenuNode.value?.server.id || 0,
    host: '',
    port: 0,
    username: '',
    password: ''
  }
}

const handleConnect = () => {
  if (contextMenuNode.value && contextMenuNode.value.server.host) {
    console.log('右键菜单连接到服务器:', contextMenuNode.value.server)
    emit('select-server', contextMenuNode.value.server)
  }
  showContextMenu.value = false
}

const handleEdit = () => {
  // 实现编辑服务器/文件夹逻辑
  if (!contextMenuNode.value) return;
  
  const server = contextMenuNode.value.server;
  editingServer.value = {
    id: server.id,
    name: server.name,
    type: server.host ? 'server' : 'folder',
    host: server.host || '',
    port: server.port || 23,
    username: server.username || '',
    password: server.password || '',
    parent_id: server.parent_id
  };
  
  showEditDialog.value = true;
  showContextMenu.value = false;
}

// 获取节点的所有子节点ID（包括自身）
const getAllChildrenIds = (nodeId: number): number[] => {
  const ids: number[] = [nodeId];
  
  const findChildren = (nodes: ServerTreeNode[]) => {
    for (const node of nodes) {
      if (node.server.parent_id === nodeId) {
        ids.push(node.server.id);
        if (node.children && node.children.length > 0) {
          findChildren(node.children);
        }
      }
    }
  };
  
  // 从整个树中查找子节点
  findChildren(serverTree.value);
  return ids;
}

// 获取可选的目标文件夹（排除自身及子文件夹）
const availableFolders = computed(() => {
  if (!moveTarget.value.id) return folders.value;
  
  // 获取当前节点的所有子节点ID
  const excludeIds = getAllChildrenIds(moveTarget.value.id);
  
  // 过滤掉自身及所有子节点
  return folders.value.filter(folder => !excludeIds.includes(folder.server.id));
});

const handleMove = () => {
  if (!contextMenuNode.value) return;
  
  // 如果是根目录的服务器
  const isRootServer = contextMenuNode.value.server.parent_id === 0;
  
  // 设置移动目标信息
  moveTarget.value = {
    id: contextMenuNode.value.server.id,
    parentId: isRootServer ? 0 : contextMenuNode.value.server.parent_id, // 确保根目录显示正确
    originalParentId: contextMenuNode.value.server.parent_id
  };
  
  showMoveDialog.value = true;
  showContextMenu.value = false;
}

const cancelMove = () => {
  showMoveDialog.value = false;
  // 清空移动目标信息
  setTimeout(() => {
    moveTarget.value = {
      id: 0,
      parentId: 0,
      originalParentId: 0
    };
  }, 200);
}

// 添加通用的服务器操作函数 - 创建服务器并删除旧服务器（用于移动和更新操作）
const createAndReplaceServer = async (newServerData: any, oldServerId: number, successMessage: string) => {
  try {
    // 1. 创建新服务器
    console.log('创建新服务器:', newServerData);
    const createResponse = await fetch('http://localhost:8080/servers', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Accept': 'application/json'
      },
      body: JSON.stringify(newServerData)
    });
    
    if (!createResponse.ok) {
      const errorText = await createResponse.text();
      console.error('创建服务器失败:', errorText);
      ElMessage.error(`操作失败: 无法创建服务器 (${createResponse.status})`);
      return false;
    }
    
    const createdServer = await createResponse.json();
    console.log('新服务器创建成功:', createdServer);
    
    // 2. 删除原始服务器
    console.log(`删除原始服务器: ${oldServerId}`);
    const deleteResponse = await fetch(`http://localhost:8080/servers/${oldServerId}`, {
      method: 'DELETE'
    });
    
    if (!deleteResponse.ok) {
      const errorText = await deleteResponse.text();
      console.error('删除原始服务器失败:', errorText);
      ElMessage.warning('注意: 服务器已在新位置创建，但无法删除原位置的服务器，可能会出现重复项');
    } else {
      console.log('原始服务器删除成功');
    }
    
    ElMessage.success(successMessage);
    return true;
  } catch (error) {
    console.error('操作服务器过程中发生错误:', error);
    
    // 检查是否是网络错误
    if (error instanceof TypeError && error.message.includes('network')) {
      ElMessage.error('网络连接失败，请确保后端服务正在运行');
    } else {
      ElMessage.error(`操作失败: ${(error as Error).message || '未知错误'}`);
    }
    return false;
  }
}

// 修改executeMove函数使用新的通用函数
const executeMove = async () => {
  if (isSameParent.value) {
    ElMessage.info('位置未更改，无需移动');
    return;
  }
  
  isMoving.value = true; // 标记开始移动
  
  try {
    // 检查是否选择了目标文件夹
    if (moveTarget.value.id === moveTarget.value.parentId) {
      ElMessage.warning('不能移动到自身');
      isMoving.value = false;
      return;
    }
    
    // 获取当前节点的所有子节点ID
    const childrenIds = getAllChildrenIds(moveTarget.value.id);
    
    // 检查是否尝试移动到子文件夹中（防止循环引用）
    if (childrenIds.includes(moveTarget.value.parentId)) {
      ElMessage.error('不能移动到自己的子文件夹中');
      isMoving.value = false;
      return;
    }
    
    // 首先获取当前服务器/文件夹信息
    console.log(`正在获取服务器ID: ${moveTarget.value.id} 的信息`);
    const serverResponse = await fetch(`http://localhost:8080/servers/${moveTarget.value.id}`);
    
    if (!serverResponse.ok) {
      ElMessage.error(`获取服务器信息失败: ${serverResponse.status} ${serverResponse.statusText}`);
      isMoving.value = false;
      return;
    }
    
    const serverData = await serverResponse.json();
    console.log('获取到服务器信息:', serverData);
    
    // 准备新服务器数据（与原始服务器相同，但parent_id不同）
    const newServerData = {
      name: serverData.name,
      host: serverData.host || '',
      port: serverData.port || 0,
      username: serverData.username || '',
      password: serverData.password || '',
      parent_id: moveTarget.value.parentId
    };
    
    // 使用通用函数创建新服务器并删除旧服务器
    const success = await createAndReplaceServer(newServerData, moveTarget.value.id, '移动成功');
    
    if (success) {
      showMoveDialog.value = false;
      
      // 重新获取服务器列表
      await fetchServers();
      
      // 如果需要，可以展开目标文件夹
      if (moveTarget.value.parentId !== 0 && treeRef.value) {
        nextTick(() => {
          // 尝试展开目标文件夹
          try {
            const expandBtn = document.querySelector(`[data-key="${moveTarget.value.parentId}"] .el-tree-node__expand-icon`);
            if (expandBtn && !expandBtn.classList.contains('is-expanded')) {
              (expandBtn as HTMLElement).click();
            }
          } catch (err) {
            console.log('展开目标文件夹失败', err);
          }
        });
      }
    }
  } catch (error) {
    console.error('移动服务器过程中发生错误:', error);
    
    // 检查是否是网络错误
    if (error instanceof TypeError && error.message.includes('network')) {
      ElMessage.error('网络连接失败，请确保后端服务正在运行');
    } else {
      ElMessage.error(`移动服务器失败: ${(error as Error).message || '未知错误'}`);
    }
  } finally {
    isMoving.value = false; // 标记移动结束
  }
}

// 在根目录添加文件夹
const handleAddRootFolder = () => {
  showAddDialog.value = true;
  newServer.value = {
    name: '',
    type: 'folder',
    parentId: 0, // 设置为根目录
    host: '',
    port: 0,
    username: '',
    password: ''
  };
  showContextMenu.value = false;
}

// 在根目录添加服务器
const handleAddRootServer = () => {
  showAddDialog.value = true;
  newServer.value = {
    name: '',
    type: 'server',
    parentId: 0, // 设置为根目录
    host: '',
    port: 23,
    username: '',
    password: ''
  };
  showContextMenu.value = false;
}

// 关闭右键菜单
const closeContextMenu = () => {
  showContextMenu.value = false
}

// 修改updateServer函数使用新的通用函数
const updateServer = async () => {
  try {
    // 验证必填字段
    if (!editingServer.value.name.trim()) {
      ElMessage.warning('名称不能为空')
      return
    }
    
    if (editingServer.value.type === 'server' && !editingServer.value.host.trim()) {
      ElMessage.warning('主机地址不能为空')
      return
    }
    
    const serverData = {
      name: editingServer.value.name.trim(),
      parent_id: editingServer.value.parent_id,
      host: editingServer.value.type === 'server' ? editingServer.value.host.trim() : '',
      port: editingServer.value.type === 'server' ? editingServer.value.port : 0,
      username: editingServer.value.username || '',
      password: editingServer.value.password || '',
    }
    
    // 使用通用函数创建新服务器并删除旧服务器
    const success = await createAndReplaceServer(serverData, editingServer.value.id, '更新成功');
    
    if (success) {
      showEditDialog.value = false;
      // 重新获取服务器列表
      await fetchServers();
    }
  } catch (error) {
    console.error('Failed to update server:', error)
    // 检查是否是网络错误
    if (error instanceof TypeError && error.message.includes('network')) {
      ElMessage.error('网络连接失败，请确保后端服务正在运行');
    } else {
      ElMessage.error(`更新服务器失败: ${(error as Error).message || '未知错误'}`);
    }
  }
}

// 生命周期钩子
onMounted(() => {
  document.addEventListener('click', closeContextMenu)
  fetchServers()
  
  // 添加主题变化监听
  window.addEventListener('storage', updateTheme)
  window.addEventListener('theme-change', handleThemeChange as EventListener)
  
  // 初始化主题
  updateTheme()
})

onUnmounted(() => {
  document.removeEventListener('click', closeContextMenu)
  
  // 移除主题变化监听
  window.removeEventListener('storage', updateTheme)
  window.removeEventListener('theme-change', handleThemeChange as EventListener)
})
</script>

<style scoped>
.server-tree-container {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: var(--bg-color);
  border-right: 1px solid var(--border-color);
}

.toolbar {
  padding: 12px;
  display: flex;
  gap: 8px;
  align-items: center;
  border-bottom: 1px solid var(--border-color);
  background: var(--panel-bg);
  backdrop-filter: blur(8px);
}

.toolbar-btn {
  padding: 8px;
  border-radius: 8px;
  transition: all 0.3s ease;
  background-color: var(--primary-color);
  color: white;
}

.toolbar-btn:hover {
  transform: translateY(-1px);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.2);
  opacity: 0.9;
}

.search-input {
  flex: 1;
  max-width: 200px;
  
  :deep(.el-input__wrapper) {
    border-radius: 8px;
    background-color: var(--bg-color);
    box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
  }
  
  :deep(.el-input__inner) {
    height: 32px;
    font-size: 14px;
    color: var(--text-color);
  }
}

.tree-container {
  flex: 1;
  overflow-y: auto;
  padding: 12px;
  background-color: var(--bg-color);
  position: relative;
  min-height: 100px; /* 确保即使没有内容，也有足够的高度可以点击 */
  cursor: context-menu; /* 提示用户此区域可以右键 */
  scrollbar-width: none; /* Firefox */
  -ms-overflow-style: none; /* IE and Edge */
}

.tree-container::after {
  content: '';
  display: block;
  min-height: 100px; /* 确保滚动区域底部也有空间可点击 */
}

.tree-content {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 50px; /* 确保树形组件有最小高度 */
}

.custom-tree-node {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 0;
  color: var(--text-color);
  cursor: pointer;
}

.custom-tree-node[data-is-server="true"] {
  cursor: pointer;
}

.custom-tree-node[data-is-folder="true"] {
  cursor: pointer;
  font-weight: 500;
}

.custom-tree-node[data-is-server="true"]:hover {
  text-decoration: underline;
  color: var(--primary-color);
}

.custom-tree-node[data-is-folder="true"]:hover {
  color: var(--primary-color);
  background-color: var(--hover-color);
  border-radius: 4px;
}

.custom-tree-node[data-is-server="true"]:active {
  transform: scale(0.98);
  transition: transform 0.1s ease;
}

.custom-tree-node[data-is-folder="true"]:active {
  transform: scale(0.98);
  transition: transform 0.1s ease;
  background-color: var(--active-color);
}

@keyframes pulse {
  0% { transform: scale(1); }
  50% { transform: scale(0.95); }
  100% { transform: scale(1); }
}

.node-pulse {
  animation: pulse 0.3s ease;
}

@keyframes folder-pulse {
  0% { transform: scale(1); }
  50% { transform: scale(0.98); background-color: var(--hover-color); }
  100% { transform: scale(1); }
}

.folder-pulse {
  animation: folder-pulse 0.3s ease;
}

.node-icon {
  font-size: 16px;
}

.node-label {
  font-size: 14px;
}

.context-menu {
  position: fixed;
  background: var(--bg-color);
  border-radius: 8px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  padding: 4px;
  z-index: 1000;
  min-width: 160px;
  border: 1px solid var(--border-color);
}

.context-menu-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  cursor: pointer;
  border-radius: 4px;
  transition: all 0.2s ease;
  color: var(--text-color);
  
  &:hover {
    background: var(--hover-color);
    color: var(--primary-color);
  }
  
  .el-icon {
    font-size: 16px;
  }
}

/* 自定义滚动条样式 - 完全隐藏 */
::-webkit-scrollbar {
  width: 0;
  height: 0;
  display: none; /* 兼容某些浏览器 */
}

::-webkit-scrollbar-track {
  background: transparent;
}

::-webkit-scrollbar-thumb {
  background: transparent;
}

::-webkit-scrollbar-thumb:hover {
  background: transparent;
}

/* 确保Firefox也隐藏滚动条 */
/* .tree-container {
  scrollbar-width: none; 
  -ms-overflow-style: none; 
} */

/* Element Plus 树形控件样式优化 */
:deep(.el-tree) {
  background: transparent;
  color: var(--text-color);
  
  .el-tree-node__content {
    border-radius: 6px;
    height: 36px;
    color: var(--text-color);
    
    &:hover {
      background: var(--hover-color);
    }
    
    &.is-current {
      background: var(--active-color);
      color: var(--primary-color);
    }
  }
}

.move-dialog-content {
  padding: 10px 0;
}

.move-source-info {
  margin-bottom: 20px;
  display: flex;
  align-items: center;
  padding: 8px 12px;
  background-color: var(--hover-color);
  border-radius: 4px;
}

.source-label {
  font-weight: bold;
  margin-right: 10px;
  color: var(--text-color);
}

.source-value {
  color: var(--primary-color);
  font-weight: 500;
}

.source-type {
  font-size: 12px;
  color: var(--text-color);
  opacity: 0.7;
  margin-left: 4px;
}

.move-form {
  margin-top: 15px;
}

.full-width-select {
  width: 100%;
}

.move-warning {
  margin-top: 15px;
}

.no-target-warning {
  margin-top: 15px;
  margin-bottom: 15px;
}
</style> 