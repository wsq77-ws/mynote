<script setup>
import { ref, onMounted, onUnmounted, nextTick } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getTree, createNote, deleteNote, searchNotes, renameNote } from '../api/index.js'

const emit = defineEmits(['selectNote', 'noteCreated', 'noteDeleted', 'showNewNote'])

const treeData = ref([])
const loading = ref(false)
const contextMenu = ref(null)
const contextMenuTarget = ref(null)
const dialogVisible = ref(false)
const dialogType = ref('file') // 'file' or 'directory' or 'rename'
const dialogParentPath = ref('default')
const newName = ref('')
const searchKeyword = ref('')
const searchResults = ref([])
const searchInputRef = ref(null)
const searchDialogVisible = ref(false)
const searchLoading = ref(false)
const renameDialogVisible = ref(false)
const renameTarget = ref(null)
const renameNewName = ref('')
const treeRef = ref(null)
const expandedPaths = ref([]) // 保持目录展开状态

// 从目录树中提取所有目录路径（含 default），用于创建笔记时选择
const directoryOptions = ref([])

function collectDirectories(nodes, prefix = '', result = []) {
  for (const node of nodes) {
    if (node.type === 'directory') {
      result.push({ label: node.path, value: node.path })
      if (node.children && node.children.length) {
        collectDirectories(node.children, node.path, result)
      }
    }
  }
  return result
}

function refreshDirectoryOptions() {
  const dirs = collectDirectories(treeData.value)
  // 确保 default 始终在选项中
  if (!dirs.find((d) => d.value === 'default')) {
    dirs.unshift({ label: 'default', value: 'default' })
  }
  directoryOptions.value = dirs
}

// 加载目录树
async function loadTree() {
  // 保存当前展开状态
  if (treeRef.value) {
    expandedPaths.value = treeRef.value.getExpandedKeys()
  }

  loading.value = true
  try {
    const res = await getTree()
    if (res.data.code === 200) {
      treeData.value = res.data.data || []
      refreshDirectoryOptions()

      // 恢复展开状态
      await nextTick()
      if (treeRef.value && expandedPaths.value.length > 0) {
        expandedPaths.value.forEach(path => {
          treeRef.value.setExpanded(path, true)
        })
      }
    }
  } catch (err) {
    console.error('加载目录树失败:', err)
  } finally {
    loading.value = false
  }
}

// 选择笔记
function selectNode(node) {
  if (node.type === 'file') {
    emit('selectNote', { path: node.path, name: node.name })
  }
}

// 右键菜单
function handleContextMenu(event, node) {
  event.preventDefault()
  contextMenuTarget.value = node
  contextMenu.value = {
    x: event.clientX,
    y: event.clientY,
  }
}

function closeContextMenu() {
  contextMenu.value = null
  contextMenuTarget.value = null
}

document.addEventListener('click', closeContextMenu)

onUnmounted(() => {
  document.removeEventListener('click', closeContextMenu)
})

// 新建笔记/目录
function showNewFileDialog(parentPath = '') {
  dialogType.value = 'file'
  dialogParentPath.value = parentPath || 'default'
  newName.value = ''
  dialogVisible.value = true
  contextMenu.value = null
}

function showNewDirectoryDialog(parentPath = '') {
  dialogType.value = 'directory'
  dialogParentPath.value = parentPath || 'default'
  newName.value = ''
  dialogVisible.value = true
  contextMenu.value = null
}

async function confirmCreate() {
  if (!newName.value.trim()) {
    ElMessage.warning('请输入名称')
    return
  }

  // 确保有目标目录，默认为 default
  const targetDir = dialogParentPath.value || 'default'

  try {
    const data = {
      path: targetDir,
      name: newName.value.trim(),
      is_dir: dialogType.value === 'directory',
      content: dialogType.value === 'file' ? `# ${newName.value.trim()}\n\n` : '',
    }
    await createNote(data)
    ElMessage.success('创建成功')
    dialogVisible.value = false
    await loadTree()

    // 如果是文件，延迟后自动打开编辑（避免404）
    if (dialogType.value === 'file') {
      const notePath = `${targetDir}/${newName.value.trim()}.md`
      // 延迟500ms确保后端完成文件创建
      await new Promise(resolve => setTimeout(resolve, 1000))
      emit('noteCreated', { path: notePath, name: newName.value.trim() })
    }
  } catch (err) {
    ElMessage.error('创建失败: ' + (err.response?.data?.message || err.message))
  }
}

// 删除节点
async function deleteNode() {
  if (!contextMenuTarget.value) return
  const node = contextMenuTarget.value
  contextMenu.value = null

  try {
    await ElMessageBox.confirm(
      `确定要删除"${node.name}"吗？${node.type === 'directory' ? '目录及其所有内容将被删除。' : ''}`,
      '确认删除',
      { confirmButtonText: '确定', cancelButtonText: '取消', type: 'warning' }
    )

    await deleteNote(node.path)
    ElMessage.success('删除成功')
    emit('noteDeleted')
    await loadTree()
  } catch (err) {
    if (err !== 'cancel') {
      ElMessage.error('删除失败: ' + (err.response?.data?.message || err.message))
    }
  }
}

// 搜索功能 - 回车后执行搜索
async function handleSearch() {
  if (!searchKeyword.value.trim()) {
    searchResults.value = []
    searchDialogVisible.value = false
    return
  }
  searchDialogVisible.value = true
  searchLoading.value = true
  try {
    const res = await searchNotes(searchKeyword.value.trim())
    if (res.data.code === 200) {
      searchResults.value = res.data.data || []
    }
  } catch (err) {
    console.error('搜索失败:', err)
    searchResults.value = []
  } finally {
    searchLoading.value = false
  }
}

function selectSearchResult(result) {
  emit('selectNote', { path: result.path, name: result.name })
  searchKeyword.value = ''
  searchResults.value = []
  searchDialogVisible.value = false
}

// 聚焦搜索框
function focusSearch() {
  nextTick(() => {
    if (searchInputRef.value) {
      searchInputRef.value.focus()
    }
  })
}

// 重命名功能
function showRenameDialog() {
  if (!contextMenuTarget.value) return
  renameTarget.value = contextMenuTarget.value
  // 去掉 .md 后缀
  const name = renameTarget.value.name
  renameNewName.value = name.endsWith('.md') ? name.slice(0, -3) : name
  renameDialogVisible.value = true
  contextMenu.value = null
}

async function confirmRename() {
  if (!renameNewName.value.trim()) {
    ElMessage.warning('请输入新名称')
    return
  }

  try {
    await renameNote(renameTarget.value.path, renameNewName.value.trim())
    ElMessage.success('重命名成功')
    renameDialogVisible.value = false
    await loadTree()
    // 如果是文件，更新编辑器
    if (renameTarget.value.type === 'file') {
      const parentPath = renameTarget.value.path.substring(0, renameTarget.value.path.lastIndexOf('/'))
      const newPath = `${parentPath}/${renameNewName.value.trim()}.md`
      emit('selectNote', { path: newPath, name: renameNewName.value.trim() })
    }
  } catch (err) {
    ElMessage.error('重命名失败: ' + (err.response?.data?.message || err.message))
  }
}

// 拖拽排序
function handleNodeDrop(draggingNode, dropNode, dropType, ev) {
  // 只允许同目录内拖拽排序（dropType === 'prev' 或 'next'）
  if (dropType !== 'prev' && dropType !== 'next') {
    ElMessage.warning('只支持同目录内排序')
    return
  }

  // 检查是否在同一父节点下
  const dragParent = draggingNode.parent?.data?.path || draggingNode.data?.path?.substring(0, draggingNode.data?.path?.lastIndexOf('/'))
  const dropParent = dropNode.parent?.data?.path || dropNode.data?.path?.substring(0, dropNode.data?.path?.lastIndexOf('/'))

  if (dragParent !== dropParent) {
    ElMessage.warning('只能同目录内排序')
    return
  }

  // 这里可以调用后端API更新排序，暂时只是前端展示
  // TODO: 调用 updateSort API
  ElMessage.success('排序已更新')
}

// 暴露方法给父组件
defineExpose({ focusSearch })

onMounted(() => {
  loadTree()
})
</script>

<template>
  <div class="sidebar">
    <div class="sidebar-header">
      <h2>MyNote</h2>
      <el-button type="primary" size="small" @click="showNewFileDialog('')">
        <el-icon><Plus /></el-icon> 新建
      </el-button>
    </div>

    <!-- 搜索框 -->
    <div class="search-area" style="padding: 8px 12px; border-bottom: 1px solid #e4e7ed;">
      <el-input
        ref="searchInputRef"
        v-model="searchKeyword"
        placeholder="搜索笔记..."
        :prefix-icon="Search"
        clearable
        @keyup.enter="handleSearch"
      />
    </div>

    <div class="sidebar-content">
      <el-tree
        ref="treeRef"
        v-loading="loading"
        :data="treeData"
        :props="{ children: 'children', label: 'name' }"
        node-key="path"
        :highlight-current="true"
        :expand-on-click-node="false"
        draggable
        @node-click="selectNode"
        @node-contextmenu="handleContextMenu"
        @node-drop="handleNodeDrop"
      >
        <template #default="{ node, data }">
          <span class="tree-item">
            <el-icon v-if="data.type === 'directory'">
              <FolderOpened />
            </el-icon>
            <el-icon v-else>
              <Document />
            </el-icon>
            <span>{{ data.name }}</span>
          </span>
        </template>
      </el-tree>
    </div>

    <!-- 右键菜单 -->
    <div
      v-if="contextMenu"
      class="context-menu"
      :style="{ left: contextMenu.x + 'px', top: contextMenu.y + 'px' }"
    >
      <div
        v-if="contextMenuTarget?.type === 'directory'"
        class="menu-item"
        @click="showNewFileDialog(contextMenuTarget.path)"
      >
        <el-icon><Plus /></el-icon> 新建笔记
      </div>
      <div
        v-if="contextMenuTarget?.type === 'directory'"
        class="menu-item"
        @click="showNewDirectoryDialog(contextMenuTarget.path)"
      >
        <el-icon><FolderAdd /></el-icon> 新建目录
      </div>
      <div class="menu-item" @click="showRenameDialog">
        <el-icon><Edit /></el-icon> 重命名
      </div>
      <div class="menu-item danger" @click="deleteNode">
        <el-icon><Delete /></el-icon> 删除
      </div>
    </div>

    <!-- 新建对话框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="dialogType === 'file' ? '新建笔记' : '新建目录'"
      width="400px"
      :close-on-click-modal="false"
    >
      <el-form @submit.prevent="confirmCreate">
        <el-form-item label="所属目录">
          <el-select v-model="dialogParentPath" placeholder="选择目录" style="width: 100%">
            <el-option
              v-for="dir in directoryOptions"
              :key="dir.value"
              :label="dir.label"
              :value="dir.value"
            />
          </el-select>
        </el-form-item>
        <el-form-item :label="dialogType === 'file' ? '笔记名称' : '目录名称'">
          <el-input
            v-model="newName"
            :placeholder="dialogType === 'file' ? '请输入笔记名称（不含.md）' : '请输入目录名称'"
            @keyup.enter="confirmCreate"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="confirmCreate">确定</el-button>
      </template>
    </el-dialog>

    <!-- 重命名对话框 -->
    <el-dialog
      v-model="renameDialogVisible"
      title="重命名"
      width="400px"
      :close-on-click-modal="false"
    >
      <el-form @submit.prevent="confirmRename">
        <el-form-item label="新名称">
          <el-input
            v-model="renameNewName"
            placeholder="请输入新名称（不含.md）"
            @keyup.enter="confirmRename"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="renameDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="confirmRename">确定</el-button>
      </template>
    </el-dialog>

    <!-- 搜索结果对话框 -->
    <el-dialog
      v-model="searchDialogVisible"
      title="搜索结果"
      width="500px"
      center
      @closed="searchResults = []"
    >
      <div v-loading="searchLoading">
        <div v-if="searchResults.length > 0">
          <div
            v-for="result in searchResults"
            :key="result.path"
            class="search-result-item"
            @click="selectSearchResult(result)"
          >
            <div style="display: flex; align-items: center; gap: 8px;">
              <el-icon v-if="result.isDir"><FolderOpened /></el-icon>
              <el-icon v-else><Document /></el-icon>
              <div>
                <div style="font-weight: 500;">{{ result.name }}</div>
                <div style="font-size: 12px; color: #909399;">{{ result.path }}</div>
              </div>
            </div>
          </div>
        </div>
        <div v-else style="text-align: center; color: #909399; padding: 20px;">
          <el-icon style="font-size: 48px; margin-bottom: 8px;"><Search /></el-icon>
          <p>未匹配到关键词</p>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<style scoped>
.search-result-item {
  padding: 12px;
  cursor: pointer;
  border-bottom: 1px solid #e4e7ed;
  transition: background 0.3s;
}

.search-result-item:hover {
  background: #f5f7fa;
}

.search-result-item:last-child {
  border-bottom: none;
}

.search-result-item:hover {
  background: #f5f7fa;
}

.result-name {
  font-size: 14px;
  color: #303133;
  font-weight: 500;
}

.result-path {
  font-size: 12px;
  color: #909399;
  margin-top: 2px;
}
</style>
