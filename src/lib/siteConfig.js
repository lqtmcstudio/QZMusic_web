import { reactive, watch } from 'vue'

const STORAGE_KEY = 'qz-site-config'

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

function deepMerge(target, source) {
  const result = JSON.parse(JSON.stringify(target))
  for (const key in source) {
    if (source[key] && typeof source[key] === 'object' && !Array.isArray(source[key])) {
      result[key] = deepMerge(result[key] || {}, source[key])
    } else {
      result[key] = source[key]
    }
  }
  return result
}

function loadConfig() {
  try {
    const stored = localStorage.getItem(STORAGE_KEY)
    if (stored) return deepMerge(defaultConfig, JSON.parse(stored))
  } catch {}
  return JSON.parse(JSON.stringify(defaultConfig))
}

export const siteConfig = reactive(loadConfig())

watch(siteConfig, () => {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(siteConfig))
}, { deep: true })

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
