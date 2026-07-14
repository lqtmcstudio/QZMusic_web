<script setup>
import { computed, ref, watch } from 'vue'
import { AlertTriangle, LoaderCircle, MessageCircle, Send, Trash2, X } from 'lucide-vue-next'
import { api } from '../lib/api'
import { appStore } from '../stores/app'
import ModalShell from './ModalShell.vue'

const props = defineProps({
  open: Boolean,
  type: String,
  item: Object,
})
const emit = defineEmits(['close', 'added', 'removed'])
const comments = ref([])
const loading = ref(false)
const sending = ref(false)
const body = ref('')
const contextTarget = ref(null)
const deleteTarget = ref(null)
const banAuthor = ref(false)
const deleting = ref(false)
const menuPosition = ref({ left: 0, top: 0 })

const remaining = computed(() => appStore.state.limits.commentsRemaining ?? 10)

watch(
  () => [props.open, props.item?.id],
  async ([open]) => {
    if (!open || !props.item) return
    loading.value = true
    try {
      comments.value = await api.comments(props.type, props.item.id)
    } catch (error) {
      appStore.toast(error.message, 'error')
    } finally {
      loading.value = false
    }
  },
)

function login() {
  appStore.login(location.pathname)
}

function openContextMenu(event, comment) {
  if (!appStore.state.me?.isDeveloper) return
  contextTarget.value = comment
  menuPosition.value = {
    left: Math.min(event.clientX, window.innerWidth - 190),
    top: Math.min(event.clientY, window.innerHeight - 78),
  }
}

function openDeleteDialog() {
  deleteTarget.value = contextTarget.value
  contextTarget.value = null
  banAuthor.value = false
}

async function confirmDelete() {
  if (!deleteTarget.value || deleting.value) return
  deleting.value = true
  try {
    const target = deleteTarget.value
    await api.deleteComment(props.type, props.item.id, target.id, !target.author.isDeveloper && banAuthor.value)
    comments.value = comments.value.filter((comment) => comment.id !== target.id)
    deleteTarget.value = null
    emit('removed', props.item.id)
    appStore.toast(banAuthor.value ? '评论已删除，作者已被社区发布封禁' : '评论已删除', 'success')
  } catch (error) {
    appStore.toast(error.message, 'error')
  } finally {
    deleting.value = false
  }
}

async function submit() {
  const value = body.value.trim()
  if (!value || sending.value) return
  sending.value = true
  try {
    const result = await api.addComment(props.type, props.item.id, value)
    comments.value.push(result.comment)
    appStore.setLimits(result.limits)
    body.value = ''
    emit('added', props.item.id)
  } catch (error) {
    appStore.toast(error.message, 'error')
  } finally {
    sending.value = false
  }
}
</script>

<template>
  <Teleport to="body">
    <Transition name="drawer">
      <div v-if="open" class="drawer-backdrop" @mousedown.self="$emit('close')">
        <aside class="comment-drawer" aria-label="评论">
          <header>
            <div>
              <span class="eyebrow">CONVERSATION</span>
              <h2>{{ item?.title }}</h2>
            </div>
            <button class="icon-button" type="button" aria-label="关闭评论" @click="$emit('close')"><X :size="20" /></button>
          </header>

          <div class="comment-list">
            <div v-if="loading" class="empty-state"><LoaderCircle class="spin" :size="24" />加载讨论中</div>
            <div v-else-if="!comments.length" class="empty-state">
              <MessageCircle :size="32" />
              <strong>还没有留言</strong>
              <span>来留下第一条认真又友善的想法吧。</span>
            </div>
            <article
              v-for="comment in comments"
              v-else
              :key="comment.id"
              class="comment-item"
              :class="{ 'comment-item--no-avatar': !comment.author.picture, 'comment-item--moderatable': appStore.state.me?.isDeveloper }"
              @contextmenu.prevent="openContextMenu($event, comment)"
            >
              <div v-if="comment.author.picture" class="avatar avatar--small">
                <img :src="comment.author.picture" alt="" />
              </div>
              <div>
                <p><strong>{{ comment.author.name }}</strong><i v-if="comment.author.isDeveloper">开发者</i><time>{{ comment.createdAt }}</time></p>
                <div>{{ comment.body }}</div>
              </div>
            </article>
          </div>

          <footer class="comment-composer">
            <template v-if="appStore.state.me">
              <p v-if="appStore.state.communityBanned" class="composer-ban-note"><AlertTriangle :size="15" />你目前无法发布功能请求和评论，请联系开发者解除封禁。</p>
              <textarea v-model="body" maxlength="500" rows="3" :disabled="appStore.state.communityBanned" aria-label="评论内容" @keydown.ctrl.enter="submit" />
              <div>
                <span>今日还可评论 {{ remaining }} 条</span>
                <button class="button button--compact" type="button" :disabled="appStore.state.communityBanned || !body.trim() || sending || remaining <= 0" @click="submit">
                  <LoaderCircle v-if="sending" class="spin" :size="16" />
                  <Send v-else :size="16" />发送
                </button>
              </div>
            </template>
            <button v-else class="button button--full" type="button" @click="login">登录后参与讨论</button>
          </footer>
        </aside>

        <template v-if="contextTarget">
          <button class="context-menu-backdrop" type="button" aria-label="关闭评论操作菜单" @click="contextTarget = null" />
          <div class="comment-context-menu" :style="menuPosition" role="menu">
            <button type="button" role="menuitem" @click="openDeleteDialog"><Trash2 :size="16" />删除评论…</button>
          </div>
        </template>
      </div>
    </Transition>
  </Teleport>

  <ModalShell :open="!!deleteTarget" title="删除这条评论？" eyebrow="DEVELOPER ACTION" @close="deleteTarget = null">
    <div v-if="deleteTarget" class="delete-dialog">
      <div class="delete-dialog__warning">
        <AlertTriangle :size="22" />
        <div>
          <strong>{{ deleteTarget.author.name }} 的评论</strong>
          <p>“{{ deleteTarget.body }}”</p>
        </div>
      </div>
      <label v-if="!deleteTarget.author.isDeveloper" class="ban-author-option">
        <input v-model="banAuthor" type="checkbox" />
        <span>
          <strong>同时封禁该用户的社区发布权限</strong>
          <small>解封前不能发布任何功能请求和评论。</small>
        </span>
      </label>
      <div class="delete-dialog__actions">
        <button class="button button--ghost" type="button" :disabled="deleting" @click="deleteTarget = null">取消</button>
        <button class="button button--danger" type="button" :disabled="deleting" @click="confirmDelete">
          <LoaderCircle v-if="deleting" class="spin" :size="17" />
          <Trash2 v-else :size="17" />
          {{ banAuthor ? '删除并封禁' : '确认删除' }}
        </button>
      </div>
    </div>
  </ModalShell>
</template>
