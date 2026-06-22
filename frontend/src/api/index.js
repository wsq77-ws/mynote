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

// 搜索笔记
export function searchNotes(keyword) {
  return api.get('/search', { params: { q: keyword } })
}

export default api
