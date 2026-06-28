<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import Sidebar from './components/Sidebar.vue'
import NoteEditor from './components/NoteEditor.vue'

const currentNote = ref(null)
const treeKey = ref(0)
const noteEditorRef = ref(null)
const showNewNoteDialog = ref(false)
const sidebarRef = ref(null)
const sidebarVisible = ref(true)

function toggleSidebar() {
  sidebarVisible.value = !sidebarVisible.value
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
    </div>
  </div>
</template>
