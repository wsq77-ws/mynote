<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { getLLMConfig, updateLLMConfig, llmGenerate } from '../api/index.js'

const emit = defineEmits(['insert-content', 'create-note'])

const activeTab = ref('generate')

// 配置表单状态
const config = ref({
  provider: 'openai-compatible',
  api_key: '',
  base_url: '',
  model: '',
  max_tokens: 512,
  temperature: 0.7,
  system_prompt: '',
  configured: false,
})
const configLoading = ref(false)
const configSaving = ref(false)
// 标记 api_key 是否为脱敏值回显（用户未修改时保存不传 api_key）
const apiKeyTouched = ref(false)

// 生成状态
const generatePrompt = ref('')
const generateResult = ref('')
const generating = ref(false)

// 加载配置
async function loadConfig() {
  configLoading.value = true
  try {
    const res = await getLLMConfig()
    if (res.data.code === 200) {
      config.value = { ...config.value, ...res.data.data }
      apiKeyTouched.value = false
    }
  } catch (err) {
    ElMessage.error('获取配置失败: ' + (err.response?.data?.message || err.message))
  } finally {
    configLoading.value = false
  }
}

// api_key 输入时标记已修改
function onApiKeyInput() {
  apiKeyTouched.value = true
}

// 保存配置（部分更新，脱敏值未修改时不传 api_key）
async function saveConfig() {
  configSaving.value = true
  try {
    const data = {}
    // 仅在用户实际修改 api_key 时提交（避免把脱敏值 ****1234 回写）
    if (apiKeyTouched.value) {
      data.api_key = config.value.api_key
    }
    if (config.value.base_url !== undefined) data.base_url = config.value.base_url
    if (config.value.model !== undefined) data.model = config.value.model
    if (config.value.system_prompt !== undefined) data.system_prompt = config.value.system_prompt

    // max_tokens / temperature：前端基础校验后提交
    const mt = Number(config.value.max_tokens)
    const tp = Number(config.value.temperature)
    if (!Number.isFinite(mt) || mt < 1 || mt > 8192) {
      ElMessage.warning('max_tokens 须为 1~8192 之间的整数')
      configSaving.value = false
      return
    }
    if (!Number.isFinite(tp) || tp <= 0 || tp > 2) {
      ElMessage.warning('temperature 须为 (0, 2] 之间的数值')
      configSaving.value = false
      return
    }
    data.max_tokens = Math.trunc(mt)
    data.temperature = tp

    const res = await updateLLMConfig(data)
    if (res.data.code === 200) {
      ElMessage.success('配置已保存')
      await loadConfig()
    } else {
      ElMessage.error('保存失败: ' + res.data.message)
    }
  } catch (err) {
    ElMessage.error('保存失败: ' + (err.response?.data?.message || err.message))
  } finally {
    configSaving.value = false
  }
}

// 生成笔记内容
async function handleGenerate() {
  if (!generatePrompt.value.trim()) {
    ElMessage.warning('请输入提示词')
    return
  }
  if (!config.value.configured) {
    ElMessage.warning('LLM 未配置，请先在"配置"页设置 API Key')
    activeTab.value = 'config'
    return
  }
  generating.value = true
  generateResult.value = ''
  try {
    const res = await llmGenerate(generatePrompt.value.trim())
    if (res.data.code === 200) {
      generateResult.value = res.data.data.content || ''
    } else {
      ElMessage.error(res.data.message)
    }
  } catch (err) {
    ElMessage.error('生成失败: ' + (err.response?.data?.message || err.message))
  } finally {
    generating.value = false
  }
}

// 将生成结果插入当前编辑器
function insertToEditor() {
  if (!generateResult.value) return
  emit('insert-content', generateResult.value)
  ElMessage.success('已插入到当前笔记')
}

// 将生成结果另存为新笔记
function saveAsNewNote() {
  if (!generateResult.value) return
  emit('create-note', generateResult.value)
}

onMounted(() => {
  loadConfig()
})

// 暴露刷新方法
defineExpose({ loadConfig })
</script>

<template>
  <div class="llm-panel">
    <div class="llm-panel-header">
      <span class="llm-title">
        <el-icon><MagicStick /></el-icon>
        <span>AI 助手</span>
      </span>
      <el-tag v-if="config.configured" type="success" size="small" effect="plain">已配置</el-tag>
      <el-tag v-else type="info" size="small" effect="plain">未配置</el-tag>
    </div>

    <el-tabs v-model="activeTab" class="llm-tabs">
      <!-- 生成笔记 -->
      <el-tab-pane label="生成" name="generate">
        <div class="llm-section">
          <el-input
            v-model="generatePrompt"
            type="textarea"
            :rows="5"
            placeholder="输入提示词，例如：写一篇关于 Vue3 组合式 API 的学习笔记"
            maxlength="2000"
            show-word-limit
          />
          <el-button
            type="primary"
            :loading="generating"
            :disabled="!generatePrompt.trim()"
            @click="handleGenerate"
            style="margin-top: 8px; width: 100%"
          >
            <el-icon v-if="!generating"><Promotion /></el-icon>
            {{ generating ? '生成中...' : '生成内容' }}
          </el-button>

          <div v-if="generateResult" class="generate-result">
            <div class="result-header">
              <span>生成结果</span>
              <div style="display: flex; gap: 6px">
                <el-button size="small" @click="insertToEditor">插入到当前笔记</el-button>
                <el-button size="small" type="primary" @click="saveAsNewNote">另存为新笔记</el-button>
              </div>
            </div>
            <div class="result-content">{{ generateResult }}</div>
          </div>
          <div v-else-if="!generating" class="empty-tip">
            <el-icon><ChatLineRound /></el-icon>
            <p>生成的内容将显示在这里</p>
          </div>
        </div>
      </el-tab-pane>

      <!-- 配置 -->
      <el-tab-pane label="配置" name="config">
        <div v-loading="configLoading" class="llm-section">
          <el-form label-position="top" size="small">
            <el-form-item label="API Key">
              <el-input
                v-model="config.api_key"
                type="password"
                show-password
                placeholder="sk-xxxxxxxxxxxx"
                @input="onApiKeyInput"
              />
              <div class="field-tip">脱敏显示，留空或保持 **** 开头则不修改</div>
            </el-form-item>
            <el-form-item label="Base URL">
              <el-input
                v-model="config.base_url"
                placeholder="https://api.deepseek.com"
              />
            </el-form-item>
            <el-form-item label="模型">
              <el-input
                v-model="config.model"
                placeholder="deepseek-v4-pro"
              />
            </el-form-item>
            <el-form-item label="Max Tokens">
              <el-input-number
                v-model="config.max_tokens"
                :min="1"
                :max="8192"
                :step="64"
                controls-position="right"
                style="width: 100%"
              />
              <div class="field-tip">单次最大生成 token 数，作用于补全/生成/总结（默认 512，生成长内容可调大）</div>
            </el-form-item>
            <el-form-item label="Temperature">
              <el-input-number
                v-model="config.temperature"
                :min="0.1"
                :max="2"
                :step="0.1"
                :precision="2"
                controls-position="right"
                style="width: 100%"
              />
              <div class="field-tip">采样温度 (0, 2]，值越大越随机，值越小越确定（默认 0.7）</div>
            </el-form-item>
            <el-form-item label="System Prompt">
              <el-input
                v-model="config.system_prompt"
                type="textarea"
                :rows="4"
                placeholder="系统提示词，定义模型行为"
              />
            </el-form-item>
            <el-button
              type="primary"
              :loading="configSaving"
              @click="saveConfig"
              style="width: 100%"
            >
              <el-icon v-if="!configSaving"><Check /></el-icon>
              保存配置
            </el-button>
          </el-form>
        </div>
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<style scoped>
.llm-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: #fff;
}

.llm-panel-header {
  padding: 12px 14px;
  border-bottom: 1px solid #e4e7ed;
  display: flex;
  align-items: center;
  gap: 8px;
}

.llm-title {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 14px;
  font-weight: 600;
  color: #303133;
  flex: 1;
}

.llm-tabs {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.llm-tabs :deep(.el-tabs__content) {
  flex: 1;
  overflow-y: auto;
}

.llm-section {
  padding: 12px 14px;
}

.generate-result {
  margin-top: 14px;
  border: 1px solid #e4e7ed;
  border-radius: 6px;
  overflow: hidden;
}

.result-header {
  padding: 8px 12px;
  background: #f5f7fa;
  border-bottom: 1px solid #e4e7ed;
  font-size: 13px;
  font-weight: 500;
  color: #303133;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.result-content {
  padding: 12px;
  font-size: 13px;
  line-height: 1.6;
  color: #606266;
  white-space: pre-wrap;
  word-break: break-word;
  max-height: 320px;
  overflow-y: auto;
}

.empty-tip {
  margin-top: 24px;
  text-align: center;
  color: #c0c4cc;
}

.empty-tip .el-icon {
  font-size: 32px;
  margin-bottom: 6px;
}

.empty-tip p {
  font-size: 12px;
}

.field-tip {
  font-size: 11px;
  color: #909399;
  line-height: 1.4;
  margin-top: 2px;
}
</style>
