<script setup>
import { computed } from 'vue'
import { Heart, MessageCircle, Pencil } from 'lucide-vue-next'
import { renderMarkdown } from '../lib/markdown'

const props = defineProps({ update: Object, canEdit: Boolean })
defineEmits(['like', 'comments', 'edit'])
const content = computed(() => renderMarkdown(props.update.body))
</script>

<template>
  <article class="square-card update-card">
    <div class="card-content">
      <header class="card-kicker">
        <span class="status-pill status-pill--brand"><i />开发动态</span>
        <time>{{ update.createdAt }}</time>
        <button v-if="canEdit" class="icon-button icon-button--tiny" type="button" aria-label="编辑动态" @click="$emit('edit', update)"><Pencil :size="15" /></button>
      </header>
      <h2>{{ update.title }}</h2>
      <div class="markdown-body update-markdown" v-html="content" />
      <footer class="card-footer">
        <div class="author-line">
          <div v-if="update.author.picture" class="avatar avatar--small">
            <img :src="update.author.picture" alt="" />
          </div>
          <div><strong>{{ update.author.name }}</strong><span>QZ 开发者</span></div>
        </div>
        <div class="engagement">
          <button type="button" :class="{ active: update.viewer?.liked }" @click="$emit('like', update)"><Heart :size="18" :fill="update.viewer?.liked ? 'currentColor' : 'none'" />{{ update.likeCount }}</button>
          <button type="button" @click="$emit('comments', update)"><MessageCircle :size="18" />{{ update.commentCount }}</button>
        </div>
      </footer>
    </div>
  </article>
</template>
