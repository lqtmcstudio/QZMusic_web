<script setup>
import { computed } from 'vue'
import { Heart, MessageCircle, Pencil, ThumbsDown, ThumbsUp, Trash2 } from 'lucide-vue-next'
import { renderMarkdown } from '../lib/markdown'

const props = defineProps({ blueprint: Object, canEdit: Boolean })
defineEmits(['like', 'vote', 'comments', 'edit', 'delete'])

const statusMeta = {
  request: ['待审阅', 'neutral'],
  in_progress: ['制作中', 'brand'],
  voting: ['待投票', 'purple'],
  deprecated: ['已废弃', 'muted'],
  released: ['已完成', 'success'],
}
const meta = computed(() => statusMeta[props.blueprint.status] || statusMeta.request)
const bodyHtml = computed(() => renderMarkdown(props.blueprint.body))
</script>

<template>
  <article class="square-card blueprint-card" :class="`card-status--${meta[1]}`">
    <div v-if="blueprint.images?.length" class="card-gallery" :class="`card-gallery--${Math.min(blueprint.images.length, 3)}`">
      <img v-for="url in blueprint.images.slice(0, 3)" :key="url" :src="url" alt="蓝图配图" loading="lazy" />
      <span v-if="blueprint.images.length > 3">+{{ blueprint.images.length - 3 }}</span>
    </div>

    <div class="card-content">
      <header class="card-kicker">
        <span class="status-pill" :class="`status-pill--${meta[1]}`"><i />{{ meta[0] }}</span>
        <span>{{ blueprint.kind === 'preview' ? '开发预告' : '功能请求' }}</span>
        <div v-if="canEdit" class="card-admin-actions">
          <button class="icon-button icon-button--tiny" type="button" aria-label="编辑蓝图" @click="$emit('edit', blueprint)"><Pencil :size="15" /></button>
          <button class="icon-button icon-button--tiny icon-button--danger" type="button" aria-label="删除蓝图" @click="$emit('delete', blueprint)"><Trash2 :size="15" /></button>
        </div>
      </header>
      <h2>{{ blueprint.title }}</h2>
      <div class="markdown-body card-copy" v-html="bodyHtml" />

      <div v-if="blueprint.status === 'in_progress'" class="progress-block">
        <div><span>开发进度</span><strong>{{ blueprint.progress }}%</strong></div>
        <div class="progress-track"><span :style="{ width: `${blueprint.progress}%` }" /></div>
      </div>

      <div v-if="blueprint.status === 'voting'" class="vote-row">
        <button type="button" :class="{ active: blueprint.viewer?.vote === 'want' }" @click="$emit('vote', blueprint, 'want')">
          <ThumbsUp :size="16" /> 想要 <b>{{ blueprint.votes.want }}</b>
        </button>
        <button type="button" :class="{ active: blueprint.viewer?.vote === 'dont_want' }" @click="$emit('vote', blueprint, 'dont_want')">
          <ThumbsDown :size="16" /> 不想要 <b>{{ blueprint.votes.dontWant }}</b>
        </button>
      </div>

      <footer class="card-footer">
        <div class="author-line">
          <div v-if="blueprint.author.picture" class="avatar avatar--small">
            <img :src="blueprint.author.picture" alt="" />
          </div>
          <div><strong>{{ blueprint.author.name }}</strong><time>{{ blueprint.updatedAt }}</time></div>
        </div>
        <div class="engagement">
          <button type="button" :class="{ active: blueprint.viewer?.liked }" @click="$emit('like', blueprint)"><Heart :size="18" :fill="blueprint.viewer?.liked ? 'currentColor' : 'none'" />{{ blueprint.likeCount }}</button>
          <button type="button" @click="$emit('comments', blueprint)"><MessageCircle :size="18" />{{ blueprint.commentCount }}</button>
        </div>
      </footer>
    </div>
  </article>
</template>
