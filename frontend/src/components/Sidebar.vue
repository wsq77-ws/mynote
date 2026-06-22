<script setup>
import { ref, onMounted, onUnmounted, nextTick } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getTree, createNote, deleteNote, searchNotes } from '../api/index.js'

const emit = defineEmits(['selectNote', 'noteCreated', 'noteDeleted'])

const treeData = ref([])
const loading = ref(false)
const contextMenu = ref(null)
const contextMenuTarget = ref(null)
const dialogVisible = ref(false)
const dialogType = ref('file') // 'file' or 'directory'
const dialogParentPath = ref('default')
const newName = ref('')
const newAuthor = ref('')

// 搜索相关
const searchKeyword = ref('')
const searchResults = ref([])
const searching = ref(false)

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
  loading.value = true
  try {
    const res = await getTree()
    if (res.data.code === 200) {
      treeData.value = res.data.data || []
      refreshDirectoryOptions()
    }
  } catch (err) {
    console.error('加载目录树失败:', err)
  } finally {
    loading.value = false
  }
}

// 搜索笔记
async function handleSearch() {
  const keyword = searchKeyword.value.trim()
  if (!keyword) {
    searchResults.value = []
    return
  }

  searching.value = true
  try {
    const res = await searchNotes(keyword)
    if (res.data.code === 200) {
      searchResults.value = res.data.data || []
    }
  } catch (err) {
    console.error('搜索失败:', err)
  } finally {
    searching.value = false
  }
}

// 点击搜索结果
function clickSearchResult(item) {
  if (!item.is_dir) {
    emit('selectNote', { path: item.path, name: item.name })
  }
  searchKeyword.value = ''
  searchResults.value = []
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
  newAuthor.value = ''
  dialogVisible.value = true
  contextMenu.value = null
}

function showNewDirectoryDialog(parentPath = '') {
  dialogType.value = 'directory'
  dialogParentPath.value = parentPath || 'default'
  newName.value = ''
  newAuthor.value = ''
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
  const author = newAuthor.value.trim() || 'default'

  try {
    const data = {
      path: targetDir,
      name: newName.value.trim(),
      author: author,
      is_dir: dialogType.value === 'directory',
      content: dialogType.value === 'file' ? `# ${newName.value.trim()}\n\n` : '',
    }
    await createNote(data)
    ElMessage.success('创建成功')
    dialogVisible.value = false
    newName.value = ''
    newAuthor.value = ''
    await loadTree()

    // 如果是文件，自动打开编辑
    if (dialogType.value === 'file') {
      const notePath = `${targetDir}/${newName.value.trim()}.md`
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
    <div class="search-box">
      <el-input
        v-model="searchKeyword"
        placeholder="搜索笔记..."
        :prefix-icon="'Search'"
        clearable
        @input="handleSearch"
        @focus="handleSearch"
      />
      <!-- 搜索结果下拉 -->
      <div v-if="searchResults.length > 0" class="search-results">
        <div
          v-for="item in searchResults"
          :key="item.path"
          class="search-item"
          @click="clickSearchResult(item)"
        >
          <el-icon v-if="item.is_dir"><FolderOpened /></el-icon>
          <el-icon v-else><Document /></el-icon>
          <span class="search-item-name">{{ item.name }}</span>
          <span class="search-item-path">{{ item.path }}</span>
        </div>
      </div>
    </div>

    <div class="sidebar-content">
      <el-tree
        v-loading="loading"
        :data="treeData"
        :props="{ children: 'children', label: 'name' }"
        node-key="path"
        :highlight-current="true"
        :expand-on-click-node="false"
        @node-click="selectNode"
        @node-contextmenu="handleContextMenu"
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
            <span v-if="data.author && data.author !== 'default'" class="tree-author">
              ({{ data.author }})
            </span>
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
        <el-form-item label="作者">
          <el-input
            v-model="newAuthor"
            placeholder="留空则使用 default"
            @keyup.enter="confirmCreate"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="confirmCreate">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.search-box {
  position: relative;
  padding: 0 12px 8px;
}

.search-results {
  position: absolute;
  top: 100%;
  left: 12px;
  right: 12px;
  background: var(--el-bg-color, #fff);
  border: 1px solid var(--el-border-color, #dcdfe6);
  border-radius: 4px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.1);
  max-height: 300px;
  overflow-y: auto;
  z-index: 100;
}

.search-item {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 12px;
  cursor: pointer;
  border-bottom: 1px solid var(--el-border-color-lighter, #f0f0f0);
}

.search-item:hover {
  background: var(--el-fill-color-light, #f5f7fa);
}

.search-item-name {
  font-weight: 500;
}

.search-item-path {
  color: var(--el-text-color-secondary, #909399);
  font-size: 12px;
  margin-left: auto;
}

.tree-author {
  color: var(--el-text-color-secondary, #909399);
  font-size: 12px;
  margin-left: 4px;
}
</style>
