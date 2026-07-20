<script setup>
import { onMounted, ref } from 'vue'
import { CheckCircle2, LoaderCircle, ShieldBan, Unlock } from 'lucide-vue-next'
import { api } from '../lib/api'
import { appStore } from '../stores/app'

const items = ref([])
const loading = ref(true)
const removingId = ref(null)

async function load() {
  loading.value = true
  try {
    items.value = await api.communityBans()
  } catch (error) {
    appStore.toast(error.message, 'error')
  } finally {
    loading.value = false
  }
}

async function unban(item) {
  removingId.value = item.user.id
  try {
    await api.unbanUser(item.user.id)
    items.value = items.value.filter((entry) => entry.user.id !== item.user.id)
    appStore.toast(`${item.user.name} 的社区发布权限已恢复`, 'success')
  } catch (error) {
    appStore.toast(error.message, 'error')
  } finally {
    removingId.value = null
  }
}

onMounted(load)
</script>

<template>
  <div class="ban-view shell-width">
    <section class="ban-hero">
      <div>
        <span class="eyebrow">COMMUNITY MODERATION</span>
        <h1>封禁管理</h1>
        <p>这里统一管理社区发布权限。被封禁的用户仍可浏览、登录和点赞，但在解除前不能提交功能请求或评论。</p>
      </div>
      <div class="ban-hero__mark" aria-hidden="true"><ShieldBan :size="34" /><span>{{ items.length }}</span></div>
    </section>

    <section class="ban-panel">
      <header>
        <div><strong>当前封禁</strong><span>{{ items.length }} 位用户</span></div>
        <button class="text-link" type="button" :disabled="loading" @click="load">刷新列表</button>
      </header>

      <div v-if="loading" class="empty-state"><LoaderCircle class="spin" :size="24" />正在读取封禁名单</div>
      <div v-else-if="!items.length" class="empty-state">
        <CheckCircle2 :size="34" />
        <strong>当前没有被封禁的用户</strong>
        <span>需要处理的用户会出现在这里。</span>
      </div>
      <div v-else class="ban-list">
        <article v-for="item in items" :key="item.user.id" class="ban-item">
          <div class="ban-user">
            <div v-if="item.user.picture" class="avatar ban-user__avatar"><img :src="item.user.picture" alt="" /></div>
            <div>
              <strong>{{ item.user.name }}</strong>
              <span v-if="item.user.username">@{{ item.user.username }}</span>
              <small>由 {{ item.bannedBy.name }} 于 {{ item.createdAt }} 封禁</small>
            </div>
          </div>
          <button class="button button--compact button--ghost" type="button" :disabled="removingId === item.user.id" @click="unban(item)">
            <LoaderCircle v-if="removingId === item.user.id" class="spin" :size="16" />
            <Unlock v-else :size="16" />
            解除封禁
          </button>
        </article>
      </div>
    </section>
  </div>
</template>
