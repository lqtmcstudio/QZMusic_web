<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { LoaderCircle, Plus, Send } from 'lucide-vue-next'
import CommentDrawer from '../components/CommentDrawer.vue'
import ImageUploader from '../components/ImageUploader.vue'
import ModalShell from '../components/ModalShell.vue'
import UpdateCard from '../components/UpdateCard.vue'
import { api } from '../lib/api'
import { appStore } from '../stores/app'

const items = ref([])
const loading = ref(true)
const formOpen = ref(false)
const saving = ref(false)
const editingId = ref(null)
const commentItem = ref(null)
const form = reactive({ title: '', body: '', images: [] })
const isDeveloper = computed(() => !!appStore.state.me?.isDeveloper)

async function load() {
  loading.value = true
  try {
    items.value = await api.updates()
  } catch (error) {
    appStore.toast(error.message, 'error')
  } finally {
    loading.value = false
  }
}

function openCreate() {
  Object.assign(form, { title: '', body: '', images: [] })
  editingId.value = null
  formOpen.value = true
}

function openEdit(item) {
  editingId.value = item.id
  Object.assign(form, { title: item.title, body: item.body, images: [] })
  formOpen.value = true
}

function insertImage(url) {
  const suffix = form.body && !form.body.endsWith('\n') ? '\n\n' : ''
  form.body += `${suffix}![开发动态配图](${url})\n`
}

async function save() {
  if (!form.title.trim() || !form.body.trim()) return appStore.toast('标题和正文都需要填写', 'warning')
  saving.value = true
  try {
    const payload = { title: form.title, body: form.body }
    if (editingId.value) await api.updateUpdate(editingId.value, payload)
    else await api.createUpdate(payload)
    formOpen.value = false
    await load()
    appStore.toast(editingId.value ? '动态已更新' : '动态已发布', 'success')
  } catch (error) {
    appStore.toast(error.message, 'error')
  } finally {
    saving.value = false
  }
}

function requireLogin() {
  if (appStore.state.me) return true
  appStore.login('/updates')
  return false
}

async function toggleLike(item) {
  if (!requireLogin()) return
  try {
    const result = await api.toggleLike('updates', item.id)
    item.viewer.liked = result.liked
    item.likeCount = result.count
  } catch (error) {
    appStore.toast(error.message, 'error')
  }
}

function incrementComment(id) {
  const item = items.value.find((entry) => entry.id === id)
  if (item) item.commentCount += 1
}

function decrementComment(id) {
  const item = items.value.find((entry) => entry.id === id)
  if (item) item.commentCount = Math.max(0, item.commentCount - 1)
}

onMounted(load)
</script>

<template>
  <div class="plaza-view updates-view">
    <section class="plaza-hero plaza-hero--updates shell-width">
      <div>
        <span class="eyebrow">FROM THE BUILD ROOM</span>
        <h1>开发，不只是<br /><em>完成清单。</em></h1>
        <p>这里记录新功能、修复和那些值得说出来的取舍。来自 QZ Music 开发者的第一手现场。</p>
      </div>
      <div class="code-note" aria-hidden="true">
        <span>01</span><p><b>feat</b>(player): spring motion</p>
        <span>02</span><p><b>fix</b>(lyrics): scroll position</p>
        <span>03</span><p><i>// keep it light & alive</i></p>
      </div>
    </section>

    <section class="plaza-toolbar shell-width">
      <div>
        <span class="live-dot" />
        <strong>开发手记</strong>
        <span>按发布时间排列</span>
      </div>
      <button v-if="isDeveloper" class="button button--compact" type="button" @click="openCreate"><Plus :size="17" />发布动态</button>
    </section>

    <section class="square-grid update-grid shell-width">
      <div v-if="loading" class="plaza-loading"><LoaderCircle class="spin" :size="28" /> 正在读取开发现场…</div>
      <div v-else-if="!items.length" class="plaza-loading">第一条开发动态还在路上。</div>
      <UpdateCard
        v-for="item in items"
        v-else
        :key="item.id"
        :update="item"
        :can-edit="isDeveloper"
        @like="toggleLike"
        @comments="commentItem = $event"
        @edit="openEdit"
      />
    </section>

    <ModalShell :open="formOpen" :title="editingId ? '编辑开发动态' : '发布开发动态'" eyebrow="WRITE IN MARKDOWN" wide @close="formOpen = false">
      <form class="editor-form" @submit.prevent="save">
        <label>标题<input v-model="form.title" maxlength="120" /></label>
        <label>正文<textarea v-model="form.body" maxlength="20000" rows="12" /></label>
        <div class="field-group">
          <span>插入图片</span>
          <ImageUploader v-model="form.images" :max="20" compact @uploaded="insertImage" />
          <small>上传完成后会自动插入 Markdown 图片语法。</small>
        </div>
        <div class="form-footer">
          <p>动态只对开发者开放发布，所有人都可以阅读。</p>
          <button class="button" type="submit" :disabled="saving"><LoaderCircle v-if="saving" class="spin" :size="17" /><Send v-else :size="17" />{{ editingId ? '保存修改' : '发布动态' }}</button>
        </div>
      </form>
    </ModalShell>

    <CommentDrawer :open="!!commentItem" type="updates" :item="commentItem" @close="commentItem = null" @added="incrementComment" @removed="decrementComment" />
  </div>
</template>
