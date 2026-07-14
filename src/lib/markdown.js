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
