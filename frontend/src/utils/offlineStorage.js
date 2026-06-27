/**
 * 离线存储工具
 * 当后端不可用时，将笔记内容缓存到 localStorage
 * 网络恢复后自动同步到后端
 */

const STORAGE_PREFIX = 'mynote_offline_'
const DIRTY_LIST_KEY = 'mynote_offline_dirty_list'

/**
 * 保存笔记到 localStorage
 * @param {string} path - 笔记路径
 * @param {string} content - 笔记内容
 * @param {string[]} tags - 标签列表
 */
export function saveToLocal(path, content, tags = []) {
  const key = STORAGE_PREFIX + path
  const data = {
    content,
    tags,
    savedAt: Date.now(),
  }
  localStorage.setItem(key, JSON.stringify(data))
  addToDirtyList(path)
}

/**
 * 从 localStorage 读取笔记
 * @param {string} path - 笔记路径
 * @returns {{ content: string, tags: string[], savedAt: number } | null}
 */
export function loadFromLocal(path) {
  const key = STORAGE_PREFIX + path
  const raw = localStorage.getItem(key)
  if (!raw) return null
  try {
    return JSON.parse(raw)
  } catch {
    return null
  }
}

/**
 * 删除本地缓存
 * @param {string} path - 笔记路径
 */
export function removeFromLocal(path) {
  const key = STORAGE_PREFIX + path
  localStorage.removeItem(key)
  removeFromDirtyList(path)
}

/**
 * 获取所有脏数据路径列表（有离线修改未同步的笔记）
 * @returns {string[]}
 */
export function getDirtyList() {
  const raw = localStorage.getItem(DIRTY_LIST_KEY)
  if (!raw) return []
  try {
    return JSON.parse(raw)
  } catch {
    return []
  }
}

/**
 * 添加路径到脏数据列表
 */
function addToDirtyList(path) {
  const list = getDirtyList()
  if (!list.includes(path)) {
    list.push(path)
    localStorage.setItem(DIRTY_LIST_KEY, JSON.stringify(list))
  }
}

/**
 * 从脏数据列表中移除路径
 */
function removeFromDirtyList(path) {
  const list = getDirtyList()
  const idx = list.indexOf(path)
  if (idx !== -1) {
    list.splice(idx, 1)
    localStorage.setItem(DIRTY_LIST_KEY, JSON.stringify(list))
  }
}

/**
 * 检查是否有未同步的离线数据
 * @returns {boolean}
 */
export function hasUnsyncedData() {
  return getDirtyList().length > 0
}

/**
 * 检查网络是否可用（通过 navigator.onLine）
 * @returns {boolean}
 */
export function isOnline() {
  return navigator.onLine
}
