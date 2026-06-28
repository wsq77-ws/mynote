import axios from 'axios'

const api = axios.create({
  baseURL: '/api',
  timeout: 10000,
})

// 网络抖动自动重试拦截器
// 对网络错误、超时、5xx 响应自动重试，指数退避（2s, 4s, 8s），最多 3 次
api.interceptors.response.use(undefined, async (error) => {
  const config = error.config
  if (!config || config.__noRetry) return Promise.reject(error)

  if (!config.__retryCount) config.__retryCount = 0

  const isNetworkError = !error.response
  const isTimeout = error.code === 'ECONNABORTED'
  const isServerError = error.response && error.response.status >= 500
  const isRetryable = isNetworkError || isTimeout || isServerError

  if (!isRetryable || config.__retryCount >= 3) {
    return Promise.reject(error)
  }

  config.__retryCount++
  const delay = Math.pow(2, config.__retryCount) * 1000
  await new Promise((resolve) => setTimeout(resolve, delay))
  return api.request(config)
})

// 健康检查（不参与重试，独立短超时）
export function healthCheck() {
  return api.get('/health', { timeout: 5000, __noRetry: true })
}

// 获取目录树
export function getTree(path = '') {
  const params = {}
  if (path) params.path = path
  return api.get('/tree', { params })
}

// 获取笔记内容
export function getNote(path) {
  return api.get('/note', { params: { path } })
}

// 创建笔记或目录
export function createNote(data) {
  return api.post('/note', data)
}

// 更新笔记内容
export function updateNote(path, content) {
  return api.put('/note', { content }, { params: { path } })
}

// 删除笔记或目录
export function deleteNote(path) {
  return api.delete('/note', { params: { path } })
}

// 搜索笔记（名称和内容）
export function searchNotes(query) {
  return api.get('/search', { params: { query } })
}

// 获取笔记的标签
export function getTags(path) {
  return api.get('/tags', { params: { path } })
}

// 获取笔记标签（别名）
export function getNoteTags(path) {
  return getTags(path)
}

// 添加标签
export function addTag(path, tag) {
  return api.post('/tags', { path, tag })
}

// 删除标签
export function removeTag(path, tag) {
  return api.delete('/tags', { data: { path, tag } })
}

// 更新笔记全部标签
export function updateNoteTags(path, tags) {
  // 逐个同步：先获取当前标签，再增删
  return api.put('/tags', { path, tags })
}

// 按标签搜索
export function searchByTag(tag) {
  return api.get('/tags/search', { params: { tag } })
}

// 获取所有标签
export function getAllTags() {
  return api.get('/tags/all')
}

// 重命名笔记或目录
export function renameNote(path, newName) {
  return api.put('/rename', null, { params: { path, newName } })
}

// 更新排序
export function updateSortOrder(path, sortOrder) {
  return api.post('/sort', { path, sortOrder })
}

// ===== LLM 相关 =====

// 获取 LLM 配置（api_key 脱敏）
export function getLLMConfig() {
  return api.get('/llm/config')
}

// 更新 LLM 配置（部分更新）
export function updateLLMConfig(data) {
  return api.put('/llm/config', data)
}

// 自动补全（F1，不参与重试，独立短超时）
export function llmComplete(text) {
  return api.post('/llm/complete', { text }, { timeout: 15000, __noRetry: true })
}

// 生成笔记内容（F2，长超时）
export function llmGenerate(prompt) {
  return api.post('/llm/generate', { prompt }, { timeout: 120000 })
}

// 总结所有笔记（F3，长超时）
export function llmSummarize() {
  return api.post('/llm/summarize', {}, { timeout: 180000 })
}

export default api