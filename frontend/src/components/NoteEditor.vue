<script setup>
import { ref, watch, onMounted, onUnmounted, computed, nextTick } from 'vue'
import { ElMessage } from 'element-plus'
import { getNote, updateNote, updateNoteTags, getNoteTags, healthCheck, llmComplete } from '../api/index.js'
import { saveToLocal, loadFromLocal, removeFromLocal, getDirtyList, hasUnsyncedData, isOnline } from '../utils/offlineStorage.js'
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
const tags = ref([])
const newTag = ref('')
const showTagInput = ref(false)
const tagInputRef = ref(null)
const previewOnly = ref(false)
const isOffline = ref(!navigator.onLine)
const hasLocalCache = ref(false)

// LLM 自动补全（F1）
const autoCompleteEnabled = ref(false)
const currentSuggestion = ref('')
let completeTimer = null
let completeInFlight = false
// 加载笔记后跳过首次 content 变化触发的补全
let skipNextComplete = false

// 字数统计
const stats = computed(() => {
  const text = content.value || ''
  const chineseChars = (text.match(/[\u4e00-\u9fa5]/g) || []).length
  const englishWords = (text.match(/[a-zA-Z]+/g) || []).length
  const wordCount = chineseChars + englishWords
  const lineCount = text ? text.split('\n').length : 0
  const readTime = Math.ceil(wordCount / 200)
  return { wordCount, lineCount, readTime }
})

// 网络恢复处理
// navigator.onLine 仅代表浏览器有网络连接，不代表后端可达
// 先健康检查确认服务器可用，再触发同步
async function handleOnline() {
  try {
    await healthCheck()
    isOffline.value = false
    syncOfflineData()
  } catch {
    // 服务器仍不可达，保持离线状态
    isOffline.value = true
  }
}

function handleOffline() {
  isOffline.value = true
}

onMounted(() => {
  window.addEventListener('online', handleOnline)
  window.addEventListener('offline', handleOffline)
})

onUnmounted(() => {
  window.removeEventListener('online', handleOnline)
  window.removeEventListener('offline', handleOffline)
})

// 同步离线数据到后端
async function syncOfflineData() {
  const dirtyList = getDirtyList()
  if (dirtyList.length === 0) return

  ElMessage.info(`检测到 ${dirtyList.length} 篇笔记有离线修改，正在同步...`)

  let synced = 0
  for (const path of dirtyList) {
    const localData = loadFromLocal(path)
    if (localData) {
      try {
        await updateNote(path, localData.content)
        if (localData.tags && localData.tags.length > 0) {
          await updateNoteTags(path, localData.tags)
        }
        removeFromLocal(path)
        synced++
      } catch (err) {
        console.error(`同步 ${path} 失败:`, err)
      }
    } else {
      removeFromLocal(path)
    }
  }

  if (synced > 0) {
    ElMessage.success(`已同步 ${synced} 篇离线笔记`)
    emit('saved')
    // 刷新当前笔记内容
    hasLocalCache.value = false
  }
}

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
  // 切换笔记时跳过首次 content 变化的自动补全触发，避免加载即请求
  skipNextComplete = true
  currentSuggestion.value = ''
  try {
    const res = await getNote(path)
    if (res.data.code === 200) {
      const serverContent = res.data.data.content || ''
      const serverTags = res.data.data.tags || []

      // 检查是否有离线缓存
      const localData = loadFromLocal(path)
      if (localData) {
        // 如果离线缓存比服务端更新，使用离线版本
        content.value = localData.content
        tags.value = localData.tags || serverTags
        hasLocalCache.value = true
        ElMessage.warning('检测到离线修改，已加载本地版本。网络恢复后将自动同步。')
      } else {
        content.value = serverContent
        tags.value = serverTags
        hasLocalCache.value = false
      }

      title.value = res.data.data.name
      fileInfo.value = res.data.data
    }
  } catch (err) {
    // 网络请求失败，尝试从本地加载
    const localData = loadFromLocal(path)
    if (localData) {
      content.value = localData.content
      tags.value = localData.tags || []
      title.value = path.split('/').pop().replace(/\.md$/, '')
      hasLocalCache.value = true
      isOffline.value = true
      ElMessage.warning('无法连接服务器，已加载本地缓存版本')
    } else {
      ElMessage.error('加载笔记失败: ' + (err.response?.data?.message || err.message))
    }
  }
}

// 自动保存
let saveTimer = null
// 保存去重：避免网络慢时并发保存产生竞态
let saveInFlight = false
let pendingSave = false
watch(content, () => {
  if (saveTimer) clearTimeout(saveTimer)
  saveTimer = setTimeout(() => {
    saveContent()
  }, 2000)
})

async function saveContent() {
  if (!props.note?.path) return
  // 已有保存请求在途，标记待保存，等当前保存完成后再次保存最新内容
  if (saveInFlight) {
    pendingSave = true
    return
  }
  saveInFlight = true
  saving.value = true
  try {
    await updateNote(props.note.path, content.value)
    // 保存成功，清除本地缓存
    removeFromLocal(props.note.path)
    hasLocalCache.value = false
    emit('saved')
  } catch (err) {
    // 保存失败，写入本地缓存
    console.error('保存到服务器失败，已保存到本地:', err)
    saveToLocal(props.note.path, content.value, tags.value)
    hasLocalCache.value = true
    isOffline.value = true
  } finally {
    saveInFlight = false
    saving.value = false
    // 保存期间内容又变了，再保存一次最新内容
    if (pendingSave) {
      pendingSave = false
      saveContent()
    }
  }
}

async function manualSave() {
  if (saveTimer) clearTimeout(saveTimer)
  await saveContent()
  await saveTags()

  if (isOffline.value || hasLocalCache.value) {
    ElMessage.success('已保存到本地（离线模式），网络恢复后自动同步')
  } else {
    ElMessage.success('已保存')
  }
}

async function saveTags() {
  if (!props.note?.path) return
  try {
    await updateNoteTags(props.note.path, tags.value)
  } catch (err) {
    // 标签保存失败，也写入本地缓存
    console.error('保存标签失败:', err)
    saveToLocal(props.note.path, content.value, tags.value)
  }
}

// 添加标签
async function addTag() {
  const tag = newTag.value.trim()
  if (tag && !tags.value.includes(tag)) {
    tags.value.push(tag)
    await saveTags()
  }
  newTag.value = ''
  showTagInput.value = false
}

// 标签颜色轮换
const tagTypes = ['primary', 'success', 'warning', 'danger', 'info']
function getTagType(tag, index) {
  return tagTypes[index % tagTypes.length]
}

// 显示标签输入框并自动聚焦
async function showTagInputWithFocus() {
  showTagInput.value = true
  await nextTick()
  tagInputRef.value?.focus()
}

// 删除标签
function removeTag(index) {
  tags.value.splice(index, 1)
  saveTags()
}

// 切换预览模式
function togglePreview() {
  previewOnly.value = !previewOnly.value
}

// 手动触发同步
async function triggerSync() {
  if (!isOnline()) {
    ElMessage.warning('当前无网络连接，无法同步')
    return
  }
  await syncOfflineData()
}

// 在编辑器末尾插入内容（用于 LLM 生成结果插入）
// 触发自动保存流程，保证内容持久化
function insertContent(text) {
  if (!text) return
  const sep = content.value && !content.value.endsWith('\n') ? '\n\n' : ''
  content.value = content.value + sep + text
}

// ===== LLM 自动补全（F1）=====
// 监听内容变化，3s 防抖后以最近 100 字符为上下文请求补全
watch(content, (val) => {
  // 切换笔记加载的首次变化不触发补全
  if (skipNextComplete) {
    skipNextComplete = false
    return
  }
  // 关闭补全 / 内容为空 / 上一次请求在途 / 有未接受建议时均不触发
  if (!autoCompleteEnabled.value) return
  if (completeInFlight) return
  if (currentSuggestion.value) return
  if (!val || val.length < 5) return

  if (completeTimer) clearTimeout(completeTimer)
  completeTimer = setTimeout(() => {
    fetchSuggestion()
  }, 3000)
})

async function fetchSuggestion() {
  const tail = content.value.slice(-100) // 最近 100 字符
  if (!tail.trim()) return
  completeInFlight = true
  try {
    const res = await llmComplete(tail)
    if (res.data.code === 200) {
      const suggestion = (res.data.data.suggestion || '').trim()
      // 仅当内容未在请求期间变化时展示建议
      if (suggestion && content.value.slice(-100) === tail) {
        currentSuggestion.value = suggestion
      }
    }
    // 失败静默忽略（设计 15.5 离线降级）
  } catch (err) {
    // 自动补全失败不影响编辑，静默处理
    console.debug('[LLMComplete] 失败:', err)
  } finally {
    completeInFlight = false
  }
}

// 接受建议：追加到内容末尾（补全基于末尾上下文，追加符合预期）
function acceptSuggestion() {
  if (!currentSuggestion.value) return
  content.value = content.value + currentSuggestion.value
  currentSuggestion.value = ''
}

// 拒绝建议
function dismissSuggestion() {
  currentSuggestion.value = ''
}

// 切换 AI 补全开关
function onToggleAutoComplete(val) {
  if (!val) {
    // 关闭时清理待发请求与当前建议
    if (completeTimer) {
      clearTimeout(completeTimer)
      completeTimer = null
    }
    currentSuggestion.value = ''
  }
}

// 建议显示期间监听 Tab/Esc
function handleSuggestionKeydown(e) {
  if (!currentSuggestion.value) return
  if (e.key === 'Tab') {
    e.preventDefault()
    acceptSuggestion()
  } else if (e.key === 'Escape') {
    e.preventDefault()
    dismissSuggestion()
  }
}

onMounted(() => {
  window.addEventListener('keydown', handleSuggestionKeydown)
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleSuggestionKeydown)
})

// 暴露方法给父组件
defineExpose({ manualSave, togglePreview, triggerSync, insertContent })
</script>

<template>
  <div class="editor-container" style="display: flex; flex-direction: column; height: 100%">
    <div class="editor-header">
      <h3>
        {{ title || '未命名笔记' }}
        <el-tag v-if="isOffline" type="danger" size="small" style="margin-left: 8px;">离线</el-tag>
        <el-tag v-else-if="hasLocalCache" type="warning" size="small" style="margin-left: 8px;">待同步</el-tag>
      </h3>
      <div style="display: flex; align-items: center; gap: 8px;">
        <el-tooltip
          :content="autoCompleteEnabled ? 'AI 补全已开启（3s 防抖，Tab 接受 / Esc 拒绝）' : '开启 AI 补全'"
          placement="bottom"
        >
          <el-switch
            v-model="autoCompleteEnabled"
            inline-prompt
            active-text="AI"
            size="small"
            @change="onToggleAutoComplete"
          />
        </el-tooltip>
        <el-button
          v-if="hasLocalCache && !isOffline"
          type="warning"
          size="small"
          @click="triggerSync"
        >
          <el-icon><RefreshRight /></el-icon> 同步
        </el-button>
        <el-tag v-if="saving" type="info" size="small">保存中...</el-tag>
        <el-button type="primary" size="small" @click="manualSave" :disabled="saving">
          <el-icon><Check /></el-icon> 保存
        </el-button>
      </div>
    </div>

    <!-- 标签区域 -->
    <div class="tags-area" style="padding: 8px 12px; border-bottom: 1px solid #e4e7ed;">
      <div style="display: flex; align-items: center; flex-wrap: wrap; gap: 8px;">
        <span style="font-size: 13px; color: #606266;">
          <el-icon style="vertical-align: middle;"><PriceTag /></el-icon>
          标签:
        </span>
        <el-tag
          v-for="(tag, index) in tags"
          :key="index"
          closable
          :type="getTagType(tag, index)"
          effect="light"
          @close="removeTag(index)"
          style="margin-right: 4px;"
        >
          {{ tag }}
        </el-tag>
        <el-input
          v-if="showTagInput"
          v-model="newTag"
          ref="tagInputRef"
          placeholder="输入标签后回车添加"
          size="small"
          style="width: 150px;"
          @keyup.enter="addTag"
          @blur="showTagInput = false"
        />
        <el-button
          v-else
          size="small"
          circle
          @click="showTagInputWithFocus"
        >
          <el-icon><Plus /></el-icon>
        </el-button>
        <span v-if="tags.length === 0 && !showTagInput" style="font-size: 12px; color: #c0c4cc;">
          暂无标签
        </span>
      </div>
    </div>

    <div class="editor-area" style="flex: 1; position: relative; overflow: hidden;">
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
        :preview="false"
        :previewOnly="previewOnly"
        style="height: 100%"
      />
      <!-- AI 补全建议条（替代内联幽灵文本，Tab 接受 / Esc 拒绝） -->
      <div v-if="currentSuggestion" class="suggestion-bar">
        <span class="suggestion-text">{{ currentSuggestion }}</span>
        <span class="suggestion-actions">
          <el-button size="small" type="primary" @click="acceptSuggestion">Tab 接受</el-button>
          <el-button size="small" @click="dismissSuggestion">Esc 拒绝</el-button>
        </span>
      </div>
    </div>

    <!-- 底部状态栏 -->
    <div class="stats-bar" style="padding: 4px 12px; border-top: 1px solid #e4e7ed; font-size: 12px; color: #909399; background: #f5f7fa; display: flex; justify-content: space-between;">
      <span>字数: {{ stats.wordCount }} | 行数: {{ stats.lineCount }} | 约 {{ stats.readTime }} 分钟</span>
      <span v-if="isOffline" style="color: #f56c6c;">离线模式 - 编辑将保存到本地</span>
      <span v-else-if="hasLocalCache" style="color: #e6a23c;">有本地未同步修改</span>
    </div>
  </div>
</template>

<style scoped>
/* AI 补全建议条：浮于编辑器右下角，灰色文本 */
.suggestion-bar {
  position: absolute;
  right: 16px;
  bottom: 16px;
  z-index: 50;
  max-width: 70%;
  background: #fff;
  border: 1px solid #d9ecff;
  border-radius: 6px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
  padding: 8px 12px;
  display: flex;
  align-items: center;
  gap: 10px;
}

.suggestion-text {
  font-size: 13px;
  color: #909399;
  white-space: pre-wrap;
  word-break: break-word;
  flex: 1;
  min-width: 0;
}

.suggestion-actions {
  display: flex;
  gap: 4px;
  flex-shrink: 0;
}
</style>
