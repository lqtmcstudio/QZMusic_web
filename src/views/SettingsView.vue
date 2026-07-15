<script setup>
import { computed } from 'vue'
import { RotateCcw, Save } from 'lucide-vue-next'
import { siteConfig, defaultConfig, resetConfig, resetPage } from '../lib/siteConfig'
import { appStore } from '../stores/app'

const isDeveloper = computed(() => !!appStore.state.me?.isDeveloper)

const pages = [
  { key: 'home', label: '首页', fields: [
    { key: 'heroBadge', label: 'Hero 徽章' },
    { key: 'heroTitle', label: '主标题（第一行）' },
    { key: 'heroTitleEm', label: '主标题（渐变高亮行）' },
    { key: 'heroSubtitle', label: '副标题', area: true },
    { key: 'heroNote', label: '底部提示' },
    { key: 'manifestoEyebrow', label: '宣言眉标' },
    { key: 'manifestoTitle', label: '宣言标题（第一行）' },
    { key: 'manifestoTitleLine2', label: '宣言标题（第二行）' },
    { key: 'manifestoBody', label: '宣言正文', area: true },
  ]},
  { key: 'blueprints', label: '蓝图页', fields: [
    { key: 'heroEyebrow', label: '眉标' },
    { key: 'heroTitle', label: '主标题（第一行）' },
    { key: 'heroTitleEm', label: '主标题（渐变高亮行）' },
    { key: 'heroSubtitle', label: '副标题', area: true },
  ]},
  { key: 'updates', label: '动态页', fields: [
    { key: 'heroEyebrow', label: '眉标' },
    { key: 'heroTitle', label: '主标题（第一行）' },
    { key: 'heroTitleEm', label: '主标题（渐变高亮行）' },
    { key: 'heroSubtitle', label: '副标题', area: true },
  ]},
  { key: 'history', label: '更新历史页', fields: [
    { key: 'heroEyebrow', label: '眉标' },
    { key: 'heroTitle', label: '主标题（第一行）' },
    { key: 'heroTitleEm', label: '主标题（渐变高亮行）' },
    { key: 'heroSubtitle', label: '副标题', area: true },
  ]},
]

function saveAll() { appStore.toast('配置已保存（本地存储）', 'success') }
function handleResetPage(pageKey) { resetPage(pageKey); appStore.toast(`${pages.find(p => p.key === pageKey)?.label} 已恢复默认`, 'success') }
function handleResetAll() { resetConfig(); appStore.toast('所有页面已恢复默认文案', 'success') }
</script>

<template>
  <div class="settings-view shell-width">
    <section class="settings-hero">
      <div>
        <span class="eyebrow">SITE CONFIGURATION</span>
        <h1>页面文案<br /><em>个性化配置</em></h1>
        <p>自定义每个页面的标题、副标题和宣传文案。修改会实时保存到本地，刷新页面后仍然生效。</p>
      </div>
      <div class="settings-actions">
        <button class="button button--ghost" type="button" @click="handleResetAll"><RotateCcw :size="16" /> 恢复全部默认</button>
        <button class="button" type="button" @click="saveAll"><Save :size="16" /> 保存配置</button>
      </div>
    </section>

    <div v-if="!isDeveloper" class="settings-notice">
      <p>配置功能当前对所有登录用户开放。修改仅保存在你的浏览器本地，不会影响其他用户。</p>
    </div>

    <section v-for="page in pages" :key="page.key" class="settings-section">
      <header class="settings-section-header">
        <div>
          <strong>{{ page.label }}</strong>
          <small>{{ page.fields.length }} 个可配置项</small>
        </div>
        <button class="button button--ghost button--compact" type="button" @click="handleResetPage(page.key)"><RotateCcw :size="14" /> 恢复默认</button>
      </header>
      <div class="settings-fields">
        <label v-for="field in page.fields" :key="field.key" :class="{ full: field.area }">
          <span>{{ field.label }}</span>
          <textarea v-if="field.area" v-model="siteConfig[page.key][field.key]" rows="3" />
          <input v-else v-model="siteConfig[page.key][field.key]" />
          <small v-if="defaultConfig[page.key][field.key]">默认：{{ defaultConfig[page.key][field.key] }}</small>
        </label>
      </div>
    </section>
  </div>
</template>
