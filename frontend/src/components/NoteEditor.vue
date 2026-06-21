<script setup>
import { ref, watch, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { getNote, updateNote } from '../api/index.js'
import { MdEditor } from 'md-editor-v3'
import 'md-editor-v3/lib/style.css'

const props = defineProps({
  note: {
    type: Object,
    required: true,
  },
})

const emit = defineEmits(['saved'])

const content = ref('')
const title = ref('')
const saving = ref(false)
const fileInfo = ref(null)

// 监听 note 变化，加载内容
watch(
  () => props.note?.path,
  async (newPath) => {
    if (newPath) {
      await loadNote(newPath)
    }
  },
  { immediate: true }
)

async function loadNote(path) {
  try {
    const res = await getNote(path)
    if (res.data.code === 200) {
      content.value = res.data.data.content || ''
      title.value = res.data.data.name
      fileInfo.value = res.data.data
    }
  } catch (err) {
    ElMessage.error('加载笔记失败: ' + (err.response?.data?.message || err.message))
  }
}

// 自动保存
let saveTimer = null
watch(content, () => {
  if (saveTimer) clearTimeout(saveTimer)
  saveTimer = setTimeout(() => {
    saveContent()
  }, 2000)
})

async function saveContent() {
  if (!props.note?.path) return
  saving.value = true
  try {
    await updateNote(props.note.path, content.value)
    emit('saved')
  } catch (err) {
    console.error('保存失败:', err)
  } finally {
    saving.value = false
  }
}

async function manualSave() {
  if (saveTimer) clearTimeout(saveTimer)
  await saveContent()
  ElMessage.success('已保存')
}
</script>

<template>
  <div class="editor-container" style="display: flex; flex-direction: column; height: 100%">
    <div class="editor-header">
      <h3>{{ title || '未命名笔记' }}</h3>
      <div>
        <el-tag v-if="saving" type="info" size="small">保存中...</el-tag>
        <el-button type="primary" size="small" @click="manualSave" :disabled="saving">
          <el-icon><Check /></el-icon> 保存
        </el-button>
      </div>
    </div>
    <div class="editor-area">
      <MdEditor
        v-model="content"
        :toolbars="[
          'bold', 'italic', 'strikeThrough', 'underline', 'sub', 'sup',
          'quote', 'unorderedList', 'orderedList', 'task', 'codeRow', 'code',
          'link', 'image', 'table',
          'revoke', 'next',
          'preview', 'catalog',
        ]"
        :showCodeRowNumber="true"
        :autoDetectCode="true"
        previewTheme="github"
        style="height: 100%"
      />
    </div>
  </div>
</template>
