export class ApiError extends Error {
  constructor(message, status, code, details) {
    super(message)
    this.status = status
    this.code = code
    this.details = details
  }
}

const SUPABASE_URL = (import.meta.env.VITE_SUPABASE_URL || 'https://backend.appmiaoda.com/projects/supabase316894448002838528').replace(/\/$/, '')
const SUPABASE_ANON_KEY = import.meta.env.VITE_SUPABASE_ANON_KEY || 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJhdXRoZW50aWNhdGVkIiwiZXhwIjoyMDk0OTgwNzI0LCJpc3MiOiJzdXBhYmFzZSIsInJvbGUiOiJhbm9uIiwic3ViIjoiYW5vbiJ9.fN4QLPpYBXVSmQuCafG-hJxbVsKZBrpaBc-B9-PG2O8'

async function request(path, options = {}) {
  const headers = new Headers(options.headers || {})
  if (options.body && !(options.body instanceof FormData)) {
    headers.set('Content-Type', 'application/json')
  }

  const response = await fetch(path, {
    credentials: 'include',
    ...options,
    headers,
    body:
      options.body && !(options.body instanceof FormData)
        ? JSON.stringify(options.body)
        : options.body,
  })

  const data = response.status === 204 ? null : await response.json().catch(() => null)
  if (!response.ok) {
    throw new ApiError(data?.message || '请求失败，请稍后再试', response.status, data?.code, data?.details)
  }
  return data
}

export const api = {
  me: () => request('/api/me'),
  blueprints: (params = {}) => request(`/api/blueprints?${new URLSearchParams(params)}`),
  createBlueprint: (body) => request('/api/blueprints', { method: 'POST', body }),
  updateBlueprint: (id, body) => request(`/api/blueprints/${id}`, { method: 'PATCH', body }),
  deleteBlueprint: (id, banAuthor = false) => request(`/api/blueprints/${id}`, { method: 'DELETE', body: { banAuthor } }),
  communityBans: () => request('/api/admin/bans'),
  unbanUser: (userId) => request(`/api/admin/bans/${userId}`, { method: 'DELETE' }),
  updates: () => request('/api/updates'),
  history: (platform, params = {}) => request(`/api/history?${new URLSearchParams({ platform, ...params })}`),
  createUpdate: (body) => request('/api/updates', { method: 'POST', body }),
  updateUpdate: (id, body) => request(`/api/updates/${id}`, { method: 'PATCH', body }),
  comments: (type, id) => request(`/api/${type}/${id}/comments`),
  addComment: (type, id, body) => request(`/api/${type}/${id}/comments`, { method: 'POST', body: { body } }),
  deleteComment: (type, id, commentId, banAuthor = false) => request(`/api/${type}/${id}/comments/${commentId}`, { method: 'DELETE', body: { banAuthor } }),
  toggleLike: (type, id) => request(`/api/${type}/${id}/like`, { method: 'POST' }),
  vote: (id, choice) => request(`/api/blueprints/${id}/vote`, { method: 'POST', body: { choice } }),
  getSiteConfig: () => request('/api/config'),
  updateSiteConfig: (config) => request('/api/config', { method: 'PUT', body: config }),
  logout: () => request('/auth/logout', { method: 'POST' }),
  async upload(file) {
    if (file.size > 25 * 1024 * 1024) {
      throw new ApiError('单张图片不能超过 25MB', 400, 'file_too_large')
    }
    const form = new FormData()
    form.append('file', file)
    const response = await fetch(`${SUPABASE_URL}/functions/v1/upload-image`, {
      method: 'POST',
      headers: { Authorization: `Bearer ${SUPABASE_ANON_KEY}` },
      body: form,
    })
    const data = await response.json().catch(() => null)
    if (!response.ok || !data?.success || !data?.image?.public_url) {
      throw new ApiError(data?.error || '图片上传失败，请稍后再试', response.status, 'upload_failed')
    }
    return { url: data.image.public_url }
  },
}
