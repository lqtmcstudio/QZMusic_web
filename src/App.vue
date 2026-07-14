<script setup>
import { onMounted } from 'vue'
import { RouterLink, RouterView } from 'vue-router'
import AppHeader from './components/AppHeader.vue'
import { appStore } from './stores/app'

onMounted(() => appStore.loadMe())
</script>

<template>
  <div class="site-shell">
    <AppHeader />
    <main>
      <RouterView v-slot="{ Component }">
        <Transition name="page" mode="out-in">
          <component :is="Component" />
        </Transition>
      </RouterView>
    </main>
    <footer class="site-footer shell-width">
      <div>
        <span class="footer-mark">QZ</span>
        <div>
          <strong>QZ Music v2</strong>
          <p>纯净、开放，也认真听见每一种声音。</p>
        </div>
      </div>
      <p>Copyright © 2025–{{ new Date().getFullYear() }} QZ Cat</p>
      <RouterLink class="footer-doc-link" to="/docs/intro">使用文档 ↗</RouterLink>
    </footer>

    <div class="toast-stack" aria-live="polite">
      <TransitionGroup name="toast">
        <div v-for="toast in appStore.state.toasts" :key="toast.id" class="toast" :class="`toast--${toast.tone}`">
          {{ toast.message }}
        </div>
      </TransitionGroup>
    </div>
  </div>
</template>
