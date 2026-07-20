import { reactive } from 'vue'
import { api } from './api'

// 各页面的默认文案（后端未配置时使用）
export const defaultConfig = {
  home: {
    heroBadge: 'QZ Music v2 正在持续生长',
    heroTitle: '让每一次播放，',
    heroTitleEm: '都有回响。',
    heroSubtitle: '一款纯净、流畅，也认真倾听每种想法的多功能音乐播放器。',
    heroNote: '你提出的下一个想法，或许正是我们正在打磨的功能。',
    manifestoEyebrow: 'BUILT WITH THE COMMUNITY',
    manifestoTitle: '播放器由代码构成，',
    manifestoTitleLine2: '体验由每个人共同完成。',
    manifestoBody: '从 Flutter 到 Jetpack Compose，从单一脚本到 NodeJS 插件系统，QZ Music 的每一次重构，都为了更轻、更自由。如今，开发过程也向你敞开。',
  },
  blueprints: {
    heroEyebrow: 'PUBLIC BLUEPRINT',
    heroTitle: '下一曲，',
    heroTitleEm: '由你参与。',
    heroSubtitle: '开发进度不再藏在提交记录里。看看正在发生什么，或把你的好点子带进来。',
  },
  updates: {
    heroEyebrow: 'FROM THE BUILD ROOM',
    heroTitle: '开发，不只是',
    heroTitleEm: '完成清单。',
    heroSubtitle: '这里记录新功能、修复，以及那些值得讲述的取舍。来自 QZ Music 开发者的第一手记录。',
  },
  history: {
    heroEyebrow: 'MASTER BRANCH CHANGELOG',
    heroTitle: '每一次提交，',
    heroTitleEm: '都算数。',
    heroSubtitle: '直接从项目仓库 master 分支整理更新轨迹，文件数量与代码增减均来自 GitHub Commit 统计。',
  },
}

// 把后端扁平的 { "home.heroBadge": "xxx" } 转成嵌套结构
function flattenToNested(flat) {
  const result = JSON.parse(JSON.stringify(defaultConfig))
  for (const [key, value] of Object.entries(flat)) {
    const [page, field] = key.split('.')
    if (page && field && result[page]) {
      result[page][field] = value
    }
  }
  return result
}

// 把嵌套结构转成扁平的 { "home.heroBadge": "xxx" }
function nestedToFlatten(nested) {
  const flat = {}
  for (const [page, fields] of Object.entries(nested)) {
    for (const [field, value] of Object.entries(fields)) {
      flat[`${page}.${field}`] = value
    }
  }
  return flat
}

export const siteConfig = reactive(JSON.parse(JSON.stringify(defaultConfig)))
let loaded = false

// 从后端加载配置（全局生效）
export async function loadSiteConfig() {
  if (loaded) return
  try {
    const data = await api.getSiteConfig()
    const nested = flattenToNested(data)
    Object.keys(nested).forEach((page) => {
      Object.keys(nested[page]).forEach((key) => {
        siteConfig[page][key] = nested[page][key]
      })
    })
  } catch {
    // 后端不可用时静默使用默认值
  } finally {
    loaded = true
  }
}

// 保存配置到后端（仅开发者）
export async function saveSiteConfig(nested) {
  const flat = nestedToFlatten(nested)
  await api.updateSiteConfig(flat)
  // 更新本地响应式对象
  const updated = flattenToNested(flat)
  Object.keys(updated).forEach((page) => {
    Object.keys(updated[page]).forEach((key) => {
      siteConfig[page][key] = updated[page][key]
    })
  })
}

export function resetConfig() {
  Object.keys(defaultConfig).forEach((page) => {
    Object.keys(defaultConfig[page]).forEach((key) => {
      siteConfig[page][key] = defaultConfig[page][key]
    })
  })
}

export function resetPage(page) {
  if (!defaultConfig[page]) return
  Object.keys(defaultConfig[page]).forEach((key) => {
    siteConfig[page][key] = defaultConfig[page][key]
  })
}
