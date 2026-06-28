<script setup>
import { ref, onMounted, onUnmounted, nextTick } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getTree, createNote, deleteNote, searchNotes, renameNote, llmSummarize } from '../api/index.js'

const emit = defineEmits(['selectNote', 'noteCreated', 'noteDeleted', 'showNewNote', 'summarize-done'])

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
const summarizing = ref(false)

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
      // 默认展开所有目录
      expandedPaths.value = collectDirectories(treeData.value).map((d) => d.value)
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

// 节点展开/收起时记录状态
function handleNodeExpand(data) {
  if (!expandedPaths.value.includes(data.path)) {
    expandedPaths.value.push(data.path)
  }
}

function handleNodeCollapse(data) {
  const index = expandedPaths.value.indexOf(data.path)
  if (index > -1) {
    expandedPaths.value.splice(index, 1)
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
    // 仅移除本地节点，不触发整树刷新（整树刷新仅在新建文档时进行）
    treeRef.value?.remove(node.path)
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

// 判断标签是否匹配搜索关键词（高亮匹配的标签）
function isTagMatched(tag, keyword) {
  if (!keyword) return false
  return tag.toLowerCase().includes(keyword.toLowerCase())
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

    // 仅更新本地节点数据，不触发整树刷新（整树刷新仅在新建文档时进行）
    const oldPath = renameTarget.value.path
    const parentPath = oldPath.substring(0, oldPath.lastIndexOf('/'))
    const newNameTrimmed = renameNewName.value.trim()
    const newPath = renameTarget.value.type === 'file'
      ? `${parentPath}/${newNameTrimmed}.md`
      : `${parentPath}/${newNameTrimmed}`

    const treeNode = treeRef.value?.getNode(oldPath)
    if (treeNode) {
      treeNode.data.name = newNameTrimmed
      treeNode.data.path = newPath
    }

    // 如果是文件，更新编辑器
    if (renameTarget.value.type === 'file') {
      emit('selectNote', { path: newPath, name: newNameTrimmed })
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

// 总结所有笔记（F3）
// 调用后端汇总所有笔记并写入 default/llm_summary.md
// 重复调用会覆盖总结文档，需二次确认（设计 15.3）
async function handleSummarize() {
  try {
    await ElMessageBox.confirm(
      '将汇总所有笔记内容生成总结文档（default/llm_summary.md）。重复调用会覆盖已有总结，是否继续？',
      'AI 总结',
      { confirmButtonText: '开始总结', cancelButtonText: '取消', type: 'info' }
    )
  } catch (action) {
    return // 用户取消
  }

  summarizing.value = true
  try {
    const res = await llmSummarize()
    if (res.data.code === 200) {
      const data = res.data.data || {}
      ElMessage.success(`总结完成（${data.note_count} 篇笔记），已生成 ${data.path}`)
      // 刷新目录树以显示 llm_summary.md
      await loadTree()
      emit('summarize-done')
    } else {
      ElMessage.error(res.data.message)
    }
  } catch (err) {
    ElMessage.error('总结失败: ' + (err.response?.data?.message || err.message))
  } finally {
    summarizing.value = false
  }
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
      <div style="display: flex; gap: 6px;">
        <el-tooltip content="AI 总结所有笔记" placement="bottom">
          <el-button
            size="small"
            :loading="summarizing"
            @click="handleSummarize"
          >
            <el-icon v-if="!summarizing"><MagicStick /></el-icon>
            <span v-if="!summarizing">总结</span>
          </el-button>
        </el-tooltip>
        <el-button type="primary" size="small" @click="showNewFileDialog('')">
          <el-icon><Plus /></el-icon> 新建
        </el-button>
      </div>
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
        :default-expanded-keys="expandedPaths"
        draggable
        @node-click="selectNode"
        @node-contextmenu="handleContextMenu"
        @node-drop="handleNodeDrop"
        @node-expand="handleNodeExpand"
        @node-collapse="handleNodeCollapse"
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
      width="600px"
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
            <div style="display: flex; align-items: flex-start; gap: 8px;">
              <el-icon v-if="result.is_dir" style="margin-top: 2px;"><FolderOpened /></el-icon>
              <el-icon v-else style="margin-top: 2px;"><Document /></el-icon>
              <div style="flex: 1; min-width: 0;">
                <div style="display: flex; align-items: center; gap: 6px; flex-wrap: wrap;">
                  <span style="font-weight: 500;">{{ result.name }}</span>
                  <el-tag
                    v-if="result.match_type === 'tag'"
                    type="success"
                    size="small"
                    effect="dark"
                  >标签匹配</el-tag>
                  <el-tag
                    v-else-if="result.match_type === 'name'"
                    type="primary"
                    size="small"
                  >名称匹配</el-tag>
                  <el-tag
                    v-else-if="result.match_type === 'content'"
                    type="warning"
                    size="small"
                  >内容匹配</el-tag>
                </div>
                <div style="font-size: 12px; color: #909399; margin-top: 2px;">{{ result.path }}</div>
                <div v-if="result.snippet" style="font-size: 12px; color: #606266; margin-top: 4px; background: #f5f7fa; padding: 4px 6px; border-radius: 3px;">
                  {{ result.snippet }}
                </div>
                <div v-if="result.tags && result.tags.length > 0" style="margin-top: 6px; display: flex; gap: 4px; flex-wrap: wrap;">
                  <el-tag
                    v-for="tag in result.tags"
                    :key="tag"
                    size="small"
                    effect="plain"
                    :type="isTagMatched(tag, searchKeyword) ? 'success' : 'info'"
                  >{{ tag }}</el-tag>
                </div>
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
