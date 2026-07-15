<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { AlertTriangle, Filter, LoaderCircle, Plus, Send, Trash2 } from 'lucide-vue-next'
import BlueprintCard from '../components/BlueprintCard.vue'
import CommentDrawer from '../components/CommentDrawer.vue'
import ImageUploader from '../components/ImageUploader.vue'
import ModalShell from '../components/ModalShell.vue'
import { api } from '../lib/api'
import { appStore } from '../stores/app'

const filters = [
  { value: '', label: '全部' },
  { value: 'in_progress', label: '制作中' },
  { value: 'voting', label: '待投票' },
  { value: 'request', label: '功能请求' },
  { value: 'released', label: '已完成' },
]
const activeFilter = ref('')
const items = ref([])
const loading = ref(true)
const formOpen = ref(false)
const saving = ref(false)
const editingId = ref(null)
const commentItem = ref(null)
const deleteTarget = ref(null)
const banAuthor = ref(false)
const deleting = ref(false)
const form = reactive({ title: '', body: '', kind: 'request', status: 'request', progress: 0, images: [] })

const isDeveloper = computed(() => !!appStore.state.me?.isDeveloper)
const modalTitle = computed(() => (editingId.value ? '编辑蓝图' : isDeveloper.value ? '发布一份新蓝图' : '提交功能请求'))

async function load() {
  loading.value = true
  try {
    const params = activeFilter.value === 'request' ? { kind: 'request' } : activeFilter.value ? { status: activeFilter.value } : {}
    items.value = await api.blueprints(params)
  } catch (error) {
    appStore.toast(error.message, 'error')
  } finally {
    loading.value = false
  }
}

function openCreate() {
  if (!appStore.state.me) return appStore.login('/blueprints')
  if (!isDeveloper.value && appStore.state.communityBanned) return appStore.toast('你目前无法发布功能请求和评论，请联系开发者', 'warning')
  if (!isDeveloper.value && appStore.state.limits.requestsRemaining <= 0) return appStore.toast('今天的功能请求名额已经用完啦', 'warning')
  editingId.value = null
  Object.assign(form, { title: '', body: '', kind: isDeveloper.value ? 'preview' : 'request', status: isDeveloper.value ? 'in_progress' : 'request', progress: 0, images: [] })
  formOpen.value = true
}

function openDelete(item) {
  deleteTarget.value = item
  banAuthor.value = false
}

async function confirmDelete() {
  if (!deleteTarget.value) return
  deleting.value = true
  try {
    const target = deleteTarget.value
    await api.deleteBlueprint(target.id, !target.author.isDeveloper && banAuthor.value)
    items.value = items.value.filter((item) => item.id !== target.id)
    deleteTarget.value = null
    appStore.toast(banAuthor.value ? '蓝图已删除，作者已禁止继续发布' : '蓝图已删除', 'success')
  } catch (error) {
    appStore.toast(error.message, 'error')
  } finally {
    deleting.value = false
  }
}

function openEdit(item) {
  editingId.value = item.id
  Object.assign(form, { title: item.title, body: item.body, kind: item.kind, status: item.status, progress: item.progress, images: [...(item.images || [])] })
  formOpen.value = true
}

async function save() {
  if (!form.title.trim() || !form.body.trim()) return appStore.toast('标题和内容都需要填写', 'warning')
  saving.value = true
  try {
    const payload = { ...form, progress: Number(form.progress) }
    const result = editingId.value ? await api.updateBlueprint(editingId.value, payload) : await api.createBlueprint(payload)
    appStore.setLimits(result.limits)
    formOpen.value = false
    await load()
    appStore.toast(editingId.value ? '蓝图已更新' : '已经放进共创广场了', 'success')
  } catch (error) {
    appStore.toast(error.message, 'error')
  } finally {
    saving.value = false
  }
}

function requireLogin() {
  if (appStore.state.me) return true
  appStore.login('/blueprints')
  return false
}

async function toggleLike(item) {
  if (!requireLogin()) return
  try {
    const result = await api.toggleLike('blueprints', item.id)
    item.viewer.liked = result.liked
    item.likeCount = result.count
  } catch (error) {
    appStore.toast(error.message, 'error')
  }
}

async function vote(item, choice) {
  if (!requireLogin()) return
  try {
    const result = await api.vote(item.id, choice)
    item.viewer.vote = result.choice
    item.votes = result.votes
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
  <div class="plaza-view">
    <section class="plaza-hero plaza-hero--blueprint shell-width">
      <div>
        <span class="eyebrow">PUBLIC BLUEPRINT</span>
        <h1>下一曲，<br /><em>由你参与。</em></h1>
        <p>开发进度不再藏在提交记录里。看看正在发生什么，或把你的好点子带进来。</p>
      </div>
      <div class="hero-stat-grid">
        <div><strong>实时</strong><span>开发进度</span></div>
        <div><strong>2<span>/天</span></strong><span>功能请求</span></div>
        <div><strong>公开</strong><span>社区投票</span></div>
      </div>
    </section>

    <section class="plaza-toolbar shell-width">
      <div class="filter-tabs" aria-label="蓝图筛选">
        <Filter :size="17" />
        <button v-for="filter in filters" :key="filter.value" type="button" :class="{ active: activeFilter === filter.value }" @click="activeFilter = filter.value; load()">
          {{ filter.label }}
        </button>
      </div>
      <button class="button button--compact" type="button" @click="openCreate"><Plus :size="17" />{{ isDeveloper ? '发布蓝图' : '提个想法' }}</button>
    </section>

    <section class="square-grid shell-width">
      <div v-if="loading" class="plaza-loading"><LoaderCircle class="spin" :size="28" /> 正在展开蓝图…</div>
      <div v-else-if="!items.length" class="plaza-loading">这里暂时没有内容，换个筛选看看。</div>
      <BlueprintCard
        v-for="item in items"
        v-else
        :key="item.id"
        :blueprint="item"
        :can-edit="isDeveloper"
        @like="toggleLike"
        @vote="vote"
        @comments="commentItem = $event"
        @edit="openEdit"
        @delete="openDelete"
      />
    </section>

    <ModalShell :open="formOpen" :title="modalTitle" eyebrow="MAKE IT VISIBLE" wide @close="formOpen = false">
      <form class="editor-form" @submit.prevent="save">
        <div v-if="isDeveloper" class="form-split">
          <label>内容类型<select v-model="form.kind"><option value="preview">开发预告</option><option value="request">功能请求</option></select></label>
          <label>当前状态<select v-model="form.status"><option value="request">待审阅</option><option value="in_progress">制作中</option><option value="voting">待投票</option><option value="deprecated">已废弃</option><option value="released">已完成</option></select></label>
        </div>
        <label>标题<input v-model="form.title" maxlength="120" /></label>
        <label>详细说明<textarea v-model="form.body" maxlength="5000" rows="8" /></label>
        <label v-if="isDeveloper && form.status === 'in_progress'" class="range-label">
          <span>开发进度 <b>{{ form.progress }}%</b></span>
          <input v-model="form.progress" type="range" min="0" max="100" step="1" />
        </label>
        <div class="field-group"><span>配图（可选）</span><ImageUploader v-model="form.images" :max="5" /></div>
        <div class="form-footer">
          <p v-if="!isDeveloper">今日还可提交 <strong>{{ appStore.state.limits.requestsRemaining }}</strong> 个功能请求</p>
          <p v-else>状态和进度会实时显示在广场卡片上。</p>
          <button class="button" type="submit" :disabled="saving"><LoaderCircle v-if="saving" class="spin" :size="17" /><Send v-else :size="17" />{{ editingId ? '保存修改' : '发布' }}</button>
        </div>
      </form>
    </ModalShell>

    <ModalShell :open="!!deleteTarget" title="删除这份蓝图？" eyebrow="DEVELOPER ACTION" @close="deleteTarget = null">
      <div v-if="deleteTarget" class="delete-dialog">
        <div class="delete-dialog__warning">
          <AlertTriangle :size="22" />
          <div>
            <strong>{{ deleteTarget.title }}</strong>
            <p>删除后，相关评论、点赞和投票也会一并移除，且无法恢复。</p>
          </div>
        </div>
        <label v-if="!deleteTarget.author.isDeveloper" class="ban-author-option">
          <input v-model="banAuthor" type="checkbox" />
          <span>
            <strong>同时禁止 {{ deleteTarget.author.name }} 继续发布蓝图</strong>
            <small>之后可在“封禁管理”中解除；解封前不能发布功能请求和评论。</small>
          </span>
        </label>
        <div class="delete-dialog__actions">
          <button class="button button--ghost" type="button" :disabled="deleting" @click="deleteTarget = null">取消</button>
          <button class="button button--danger" type="button" :disabled="deleting" @click="confirmDelete">
            <LoaderCircle v-if="deleting" class="spin" :size="17" />
            <Trash2 v-else :size="17" />
            {{ banAuthor ? '删除并禁止发布' : '确认删除' }}
          </button>
        </div>
      </div>
    </ModalShell>

    <CommentDrawer :open="!!commentItem" type="blueprints" :item="commentItem" @close="commentItem = null" @added="incrementComment" @removed="decrementComment" />
  </div>
</template>
