import DOMPurify from 'dompurify'
import { marked } from 'marked'

marked.use({
  gfm: true,
  breaks: true,
})

export function renderMarkdown(value = '') {
  return DOMPurify.sanitize(marked.parse(value), {
    USE_PROFILES: { html: true },
    ADD_ATTR: ['target', 'rel'],
  })
}

// commit body 渲染：保留 #/##/### 原样（不解析为标题），其余 markdown 照常
export function renderCommitBody(value = '') {
  const escaped = value.replace(/^#{1,6}\s/gm, (match) => match.replace(/#/g, '\\#'))
  return DOMPurify.sanitize(marked.parse(escaped), {
    USE_PROFILES: { html: true },
    ADD_ATTR: ['target', 'rel'],
  })
}

