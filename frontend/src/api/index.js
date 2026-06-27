import axios from 'axios'

const api = axios.create({
  baseURL: '/api',
  timeout: 10000,
})

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

export default api