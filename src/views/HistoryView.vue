<script setup>
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { AlertCircle, ExternalLink, FileCode2, GitCommitHorizontal, LoaderCircle, Monitor, RefreshCw, Smartphone } from 'lucide-vue-next'
import { api } from '../lib/api'
import { appStore } from '../stores/app'
import { renderMarkdown } from '../lib/markdown'
import { siteConfig } from '../lib/siteConfig'

const platforms = [
  { id: 'android', name: 'Android', repo: 'nevodev/QZ-Music', icon: Smartphone, note: '安卓版 · 私有仓库' },
  { id: 'windows', name: 'Windows', repo: 'lqtmcstudio/QZMusic_PC', icon: Monitor, note: 'PC 版 · 公共仓库' },
]

const active = ref('android')
const items = ref([])
const sync = ref(null)
const loading = ref(true)
const loadingMore = ref(false)
const hasMore = ref(false)
let refreshTimer

async function load(append = false, silent = false) {
  if (!silent) append ? (loadingMore.value = true) : (loading.value = true)
  try {
    const result = await api.history(active.value, { limit: 50, offset: append ? items.value.length : 0 })
    items.value = append ? [...items.value, ...result.items] : result.items
    sync.value = result.sync
    hasMore.value = result.items.length === 50
  } catch (error) {
    if (!silent) appStore.toast(error.message, 'error')
  } finally {
    loading.value = false
    loadingMore.value = false
  }
}

function selectPlatform(platform) {
  if (active.value === platform) return
  active.value = platform
  items.value = []
  sync.value = null
  load()
}

function formatDate(value) {
  if (!value) return '尚未同步'
  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', hour12: false,
  }).format(new Date(value))
}

function formatNumber(value) {
  return new Intl.NumberFormat('en-US').format(value || 0)
}

onMounted(() => {
  load()
  refreshTimer = window.setInterval(() => load(false, true), 60_000)
})
onBeforeUnmount(() => window.clearInterval(refreshTimer))
</script>

<template>
  <div class="history-view shell-width">
    <section class="history-hero">
      <div>
        <span class="eyebrow">{{ siteConfig.history.heroEyebrow }}</span>
        <h1>{{ siteConfig.history.heroTitle }}<br /><em>{{ siteConfig.history.heroTitleEm }}</em></h1>
        <p>{{ siteConfig.history.heroSubtitle }}</p>
      </div>
      <div class="history-pulse" aria-hidden="true">
        <GitCommitHorizontal :size="28" />
        <span />
        <span />
        <span />
        <b>master</b>
      </div>
    </section>

    <section class="platform-switch" aria-label="选择项目平台">
      <button v-for="platform in platforms" :key="platform.id" type="button" :class="{ active: active === platform.id }" @click="selectPlatform(platform.id)">
        <component :is="platform.icon" :size="20" />
        <span><strong>{{ platform.name }}</strong><small>{{ platform.note }}</small></span>
        <i>{{ platform.repo }}</i>
      </button>
    </section>

    <section class="history-toolbar">
      <div>
        <span class="live-dot" />
        <div><strong>{{ platforms.find((item) => item.id === active)?.repo }}</strong><small>仅同步 master 分支</small></div>
      </div>
      <div class="history-sync">
        <RefreshCw :size="14" :class="{ spin: loading }" />
        <span>最近同步：{{ formatDate(sync?.lastSuccess) }}</span>
      </div>
    </section>

    <div v-if="sync?.error" class="history-notice" :class="{ 'history-notice--stale': items.length }">
      <AlertCircle :size="18" />
      <div>
        <strong>{{ items.length ? '当前展示已缓存记录' : sync.configured ? '暂时无法同步仓库' : '等待配置 GitHub API Key' }}</strong>
        <span>{{ sync.error }}</span>
      </div>
    </div>

    <section class="commit-timeline">
      <div v-if="loading" class="plaza-loading"><LoaderCircle class="spin" :size="27" /> 正在读取长期缓存…</div>
      <div v-else-if="!items.length" class="plaza-loading">缓存中还没有提交记录，服务端完成首次同步后会自动出现。</div>
      <article v-for="commit in items" v-else :key="commit.sha" class="commit-card">
        <div class="commit-rail"><span /><i /></div>
        <div class="commit-main">
          <header>
            <div class="commit-sha"><GitCommitHorizontal :size="15" />{{ commit.sha.slice(0, 7) }}</div>
            <time :datetime="commit.committedAt">{{ formatDate(commit.committedAt) }}</time>
          </header>
          <h2>{{ commit.title }}</h2>
          <div v-if="commit.body" class="commit-body markdown-body" v-html="renderMarkdown(commit.body)" />
          <footer>
            <div class="commit-author">
              <img v-if="commit.authorAvatar" :src="commit.authorAvatar" alt="" />
              <span><strong>{{ commit.authorName || commit.authorLogin || 'GitHub 用户' }}</strong><small v-if="commit.authorLogin">@{{ commit.authorLogin }}</small></span>
            </div>
            <div class="commit-stats">
              <span><FileCode2 :size="15" />{{ formatNumber(commit.filesChanged) }} 个文件</span>
              <b>+{{ formatNumber(commit.additions) }}</b>
              <i>−{{ formatNumber(commit.deletions) }}</i>
            </div>
            <a :href="commit.url" target="_blank" rel="noreferrer noopener" aria-label="在 GitHub 查看提交"><ExternalLink :size="16" /></a>
          </footer>
        </div>
      </article>
    </section>

    <button v-if="hasMore && !loading" class="history-more" type="button" :disabled="loadingMore" @click="load(true)">
      <LoaderCircle v-if="loadingMore" class="spin" :size="17" />
      <span>{{ loadingMore ? '正在读取…' : '加载更早的提交' }}</span>
    </button>
  </div>
</template>
