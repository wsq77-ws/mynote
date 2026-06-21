<script setup>
import { ref, onMounted } from 'vue'
import Sidebar from './components/Sidebar.vue'
import NoteEditor from './components/NoteEditor.vue'

const currentNote = ref(null)
const treeKey = ref(0)

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
</script>

<template>
  <div class="app-container">
    <Sidebar
      :key="treeKey"
      @select-note="handleSelectNote"
      @note-created="handleNoteCreated"
      @note-deleted="handleNoteDeleted"
    />
    <div class="main-content">
      <NoteEditor
        v-if="currentNote"
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
