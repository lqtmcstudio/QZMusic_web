<script setup>
import { computed, ref } from 'vue'
import { BookOpen, ChevronRight, Search } from 'lucide-vue-next'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { renderMarkdown } from '../lib/markdown'

import intro from '../../doc/index.md?raw'
import install from '../../doc/install.md?raw'
import law from '../../doc/law.md?raw'
import pluginAbout from '../../doc/plugin/about.md?raw'
import pluginDevelop from '../../doc/plugin/develop.md?raw'
import sponsorship from '../../doc/sponsorship.md?raw'

const route = useRoute()
const router = useRouter()
const query = ref('')

const groups = [
  {
    title: 'QZ Music',
    pages: [
      { slug: 'intro', title: '项目介绍', description: '认识 QZ Music v2', source: intro },
      { slug: 'install', title: '安装教程', description: '软件与插件安装', source: install },
      { slug: 'law', title: '条款与声明', description: '服务与法律条款', source: law },
    ],
  },
  {
    title: '插件开发',
    pages: [
      { slug: 'plugin/about', title: '关于插件', description: '插件体系概览', source: pluginAbout },
      { slug: 'plugin/develop', title: '开发指南', description: 'NodeJS 插件规范', source: pluginDevelop },
    ],
  },
  {
    title: '支持项目',
    pages: [{ slug: 'sponsorship', title: '鸣谢与赞助', description: '感谢每一份支持', source: sponsorship }],
  },
]

const allPages = groups.flatMap((group) => group.pages)
const currentSlug = computed(() => {
  const pathMatch = route.params.pathMatch
  return Array.isArray(pathMatch) ? pathMatch.join('/') : pathMatch || 'intro'
})
const current = computed(() => allPages.find((page) => page.slug === currentSlug.value) || allPages[0])
const visibleGroups = computed(() => {
  const keyword = query.value.trim().toLocaleLowerCase()
  if (!keyword) return groups
  return groups
    .map((group) => ({ ...group, pages: group.pages.filter((page) => `${page.title} ${page.description}`.toLocaleLowerCase().includes(keyword)) }))
    .filter((group) => group.pages.length)
})
const currentIndex = computed(() => allPages.findIndex((page) => page.slug === current.value.slug))
const previous = computed(() => allPages[currentIndex.value - 1])
const next = computed(() => allPages[currentIndex.value + 1])
const content = computed(() => renderMarkdown(rewriteLinks(current.value.source)))

function rewriteLinks(source) {
  const replacements = [
    [/https:\/\/music\.qz\.shiqianjiang\.cn\/guide\/plugin\/develop\.html/g, '/docs/plugin/develop'],
    [/https:\/\/music\.qz\.shiqianjiang\.cn\/guide\/install\.html/g, '/docs/install'],
    [/https:\/\/music\.qz\.shiqianjiang\.cn\/guide\/law\.html/g, '/docs/law'],
    [/\/guide\/plugin\/about(?:\.md|\.html)?/g, '/docs/plugin/about'],
    [/\/guide\/plugin\/develop(?:\.md|\.html)?/g, '/docs/plugin/develop'],
    [/\/guide\/install(?:\.md|\.html)?/g, '/docs/install'],
    [/\/guide\/law(?:\.md|\.html)?/g, '/docs/law'],
    [/\/guide\/sponsorship(?:\.md|\.html)?/g, '/docs/sponsorship'],
    [/\/guide\/(?:index(?:\.md|\.html)?)?/g, '/docs/intro'],
  ]
  return replacements.reduce((value, [pattern, replacement]) => value.replace(pattern, replacement), source)
}

function handleContentClick(event) {
  const anchor = event.target.closest('a')
  if (!anchor) return
  const target = new URL(anchor.href, location.origin)
  if (target.origin === location.origin && target.pathname.startsWith('/docs/')) {
    event.preventDefault()
    router.push(target.pathname + target.search + target.hash)
  } else if (target.origin !== location.origin) {
    anchor.target = '_blank'
    anchor.rel = 'noreferrer noopener'
  }
}
</script>

<template>
  <div class="docs-view shell-width">
    <aside class="docs-sidebar">
      <div class="docs-sidebar-heading">
        <span><BookOpen :size="17" /> QZ 文档</span>
        <small>旧站内容已完整迁入</small>
      </div>
      <label class="docs-search"><Search :size="15" /><input v-model="query" aria-label="搜索文档" /></label>
      <nav aria-label="文档目录">
        <section v-for="group in visibleGroups" :key="group.title">
          <h2>{{ group.title }}</h2>
          <RouterLink v-for="page in group.pages" :key="page.slug" :to="`/docs/${page.slug}`" :class="{ active: current.slug === page.slug }">
            <span><strong>{{ page.title }}</strong><small>{{ page.description }}</small></span>
            <ChevronRight :size="15" />
          </RouterLink>
        </section>
      </nav>
    </aside>

    <section class="docs-main">
      <header class="docs-page-heading">
        <span class="eyebrow">QZ MUSIC DOCUMENTATION</span>
        <p>{{ current.description }}</p>
      </header>
      <article :key="current.slug" class="markdown-body docs-article" @click="handleContentClick" v-html="content" />
      <footer class="docs-pagination">
        <RouterLink v-if="previous" :to="`/docs/${previous.slug}`"><small>上一篇</small><strong>← {{ previous.title }}</strong></RouterLink>
        <span v-else />
        <RouterLink v-if="next" :to="`/docs/${next.slug}`"><small>下一篇</small><strong>{{ next.title }} →</strong></RouterLink>
      </footer>
    </section>
  </div>
</template>
