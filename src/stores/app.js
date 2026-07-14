import { reactive, readonly } from 'vue'
import { api } from '../lib/api'

const state = reactive({
  me: null,
  limits: { commentsRemaining: 10, requestsRemaining: 2 },
  communityBanned: false,
  authLoaded: false,
  theme: localStorage.getItem('qz-theme') || 'system',
  toasts: [],
})

function applyTheme() {
  const dark =
    state.theme === 'dark' ||
    (state.theme === 'system' && window.matchMedia('(prefers-color-scheme: dark)').matches)
  document.documentElement.classList.toggle('dark', dark)
}

applyTheme()
window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', applyTheme)

export const appStore = {
  state: readonly(state),
  async loadMe() {
    try {
      const result = await api.me()
      state.me = result.user
      state.limits = result.limits
      state.communityBanned = !!result.communityBanned
    } catch {
      state.me = null
      state.communityBanned = false
    } finally {
      state.authLoaded = true
    }
  },
  setLimits(limits) {
    if (limits) state.limits = limits
  },
  toggleTheme() {
    state.theme = document.documentElement.classList.contains('dark') ? 'light' : 'dark'
    localStorage.setItem('qz-theme', state.theme)
    applyTheme()
  },
  async logout() {
    await api.logout()
    state.me = null
    state.limits = { commentsRemaining: 10, requestsRemaining: 2 }
    state.communityBanned = false
  },
  login(returnTo = location.pathname) {
    location.href = `/auth/login?return_to=${encodeURIComponent(returnTo)}`
  },
  toast(message, tone = 'default') {
    const id = crypto.randomUUID()
    state.toasts.push({ id, message, tone })
    window.setTimeout(() => {
      const index = state.toasts.findIndex((item) => item.id === id)
      if (index >= 0) state.toasts.splice(index, 1)
    }, 3400)
  },
}
