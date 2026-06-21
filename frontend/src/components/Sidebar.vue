<script setup>
import { ref, onMounted, onUnmounted, nextTick } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getTree, createNote, deleteNote } from '../api/index.js'

const emit = defineEmits(['selectNote', 'noteCreated', 'noteDeleted'])

const treeData = ref([])
const loading = ref(false)
const contextMenu = ref(null)
const contextMenuTarget = ref(null)
const dialogVisible = ref(false)
const dialogType = ref('file') // 'file' or 'directory'
const dialogParentPath = ref('')
const newName = ref('')

// 加载目录树
async function loadTree() {
  loading.value = true
  try {
    const res = await getTree()
    if (res.data.code === 200) {
      treeData.value = res.data.data || []
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
  dialogParentPath.value = parentPath
  newName.value = ''
  dialogVisible.value = true
  contextMenu.value = null
}

function showNewDirectoryDialog(parentPath = '') {
  dialogType.value = 'directory'
  dialogParentPath.value = parentPath
  newName.value = ''
  dialogVisible.value = true
  contextMenu.value = null
}

async function confirmCreate() {
  if (!newName.value.trim()) {
    ElMessage.warning('请输入名称')
    return
  }

  try {
    const data = {
      path: dialogParentPath.value,
      name: newName.value.trim(),
      is_dir: dialogType.value === 'directory',
      content: dialogType.value === 'file' ? `# ${newName.value.trim()}\n\n` : '',
    }
    await createNote(data)
    ElMessage.success('创建成功')
    dialogVisible.value = false
    newName.value = ''
    await loadTree()

    // 如果是文件，自动打开编辑
    if (dialogType.value === 'file') {
      const notePath = dialogParentPath.value
        ? `${dialogParentPath.value}/${newName.value.trim()}.md`
        : `${newName.value.trim()}.md`
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
  </div>
</template>
