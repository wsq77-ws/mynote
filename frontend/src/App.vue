<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { ElMessage } from 'element-plus'
import { createNote } from './api/index.js'
import Sidebar from './components/Sidebar.vue'
import NoteEditor from './components/NoteEditor.vue'
import LLMPanel from './components/LLMPanel.vue'

const currentNote = ref(null)
const treeKey = ref(0)
const noteEditorRef = ref(null)
const showNewNoteDialog = ref(false)
const sidebarRef = ref(null)
const sidebarVisible = ref(true)
const llmPanelVisible = ref(false)
const llmPanelRef = ref(null)

function toggleSidebar() {
  sidebarVisible.value = !sidebarVisible.value
}

function toggleLLMPanel() {
  llmPanelVisible.value = !llmPanelVisible.value
}

function handleSelectNote(note) {
  currentNote.value = note
}

function handleNoteCreated(note) {
  treeKey.value++
  currentNote.value = note
}

function handleNoteDeleted() {
  treeKey.value++
  currentNote.value = null
}

function refreshTree() {
  treeKey.value++
}

// LLM 生成结果插入当前编辑器
function handleInsertContent(text) {
  if (!currentNote.value) {
    ElMessage.warning('请先选择一个笔记')
    return
  }
  noteEditorRef.value?.insertContent(text)
}

// LLM 生成结果另存为新笔记（写入 default 目录，遵循"未指定目录存入 default"约束）
async function handleCreateNoteFromLLM(content) {
  // 从内容首行提取标题，去除 # 前缀
  const firstLine = content.split('\n')[0] || 'LLM 生成笔记'
  const rawName = firstLine.replace(/^#+\s*/, '').trim() || 'LLM 生成笔记'
  // 仅保留安全字符，避免文件名问题
  const safeName = rawName.replace(/[\\/:*?"<>|]/g, '').slice(0, 50) || 'LLM 生成笔记'
  try {
    await createNote({
      path: 'default',
      name: safeName,
      is_dir: false,
      content: content,
    })
    ElMessage.success('已保存为新笔记')
    treeKey.value++
    currentNote.value = { path: `default/${safeName}.md`, name: safeName }
  } catch (err) {
    ElMessage.error('保存失败: ' + (err.response?.data?.message || err.message))
  }
}

// 全局快捷键
function handleGlobalKeydown(e) {
  // Ctrl+S 保存当前笔记
  if (e.ctrlKey && e.key === 's') {
    e.preventDefault()
    if (noteEditorRef.value && currentNote.value) {
      noteEditorRef.value.manualSave()
    }
  }
  // Ctrl+F 打开搜索框并聚焦
  if (e.ctrlKey && e.key === 'f') {
    e.preventDefault()
    if (sidebarRef.value) {
      sidebarRef.value.focusSearch()
    }
  }
  // Ctrl+N 新建笔记
  if (e.ctrlKey && e.key === 'n') {
    e.preventDefault()
    showNewNoteDialog.value = true
  }
  // Ctrl+B 切换侧边栏显示
  if (e.ctrlKey && e.key === 'b') {
    e.preventDefault()
    toggleSidebar()
  }
  // Ctrl+L 切换 LLM 助手面板
  if (e.ctrlKey && e.key === 'l') {
    e.preventDefault()
    toggleLLMPanel()
  }
}

onMounted(() => {
  window.addEventListener('keydown', handleGlobalKeydown)
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleGlobalKeydown)
})

function closeNewNoteDialog() {
  showNewNoteDialog.value = false
}
</script>

<template>
  <div class="app-container">
    <Sidebar
      v-show="sidebarVisible"
      ref="sidebarRef"
      :key="treeKey"
      @select-note="handleSelectNote"
      @note-created="handleNoteCreated"
      @note-deleted="handleNoteDeleted"
      @show-new-note="showNewNoteDialog = true"
      @summarize-done="refreshTree"
    />
    <div class="main-content">
      <button
        class="sidebar-toggle"
        @click="toggleSidebar"
        :title="sidebarVisible ? '隐藏侧边栏 (Ctrl+B)' : '显示侧边栏 (Ctrl+B)'"
      >
        <el-icon>
          <Fold v-if="sidebarVisible" />
          <Expand v-else />
        </el-icon>
      </button>
      <NoteEditor
        v-if="currentNote"
        ref="noteEditorRef"
        :key="currentNote.path"
        :note="currentNote"
        @saved="refreshTree"
      />
      <div v-else class="empty-state">
        <el-icon><Notebook /></el-icon>
        <p>选择一个笔记，或创建新笔记开始</p>
      </div>
      <button
        class="llm-toggle"
        :class="{ active: llmPanelVisible }"
        @click="toggleLLMPanel"
        :title="llmPanelVisible ? '隐藏 AI 助手 (Ctrl+L)' : '显示 AI 助手 (Ctrl+L)'"
      >
        <el-icon><MagicStick /></el-icon>
      </button>
    </div>
    <div v-show="llmPanelVisible" class="llm-sidebar">
      <LLMPanel
        ref="llmPanelRef"
        @insert-content="handleInsertContent"
        @create-note="handleCreateNoteFromLLM"
      />
    </div>
  </div>
</template>
