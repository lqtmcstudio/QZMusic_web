<script setup>
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import { LogIn, LogOut, Menu, Moon, Sun, X } from 'lucide-vue-next'
import logoUrl from '../../src/icon.png'
import { appStore } from '../stores/app'

const menuOpen = ref(false)
const profileOpen = ref(false)
const scrolled = ref(false)
const scrollProgress = ref(0)
const route = useRoute()
let animationFrame = 0

function syncScrollState() {
  if (animationFrame) return
  animationFrame = window.requestAnimationFrame(() => {
    const maxScroll = Math.max(0, document.documentElement.scrollHeight - window.innerHeight)
    scrolled.value = window.scrollY > 14
    scrollProgress.value = maxScroll ? Math.min(1, Math.max(0, window.scrollY / maxScroll)) : 0
    animationFrame = 0
  })
}

onMounted(() => {
  syncScrollState()
  window.addEventListener('scroll', syncScrollState, { passive: true })
  window.addEventListener('resize', syncScrollState, { passive: true })
})

onBeforeUnmount(() => {
  window.removeEventListener('scroll', syncScrollState)
  window.removeEventListener('resize', syncScrollState)
  if (animationFrame) window.cancelAnimationFrame(animationFrame)
})

async function logout() {
  profileOpen.value = false
  await appStore.logout()
  appStore.toast('已安全退出')
}
</script>

<template>
  <header
    class="site-header"
    :class="{ 'is-scrolled': scrolled }"
    :style="{ '--page-progress': scrollProgress, '--page-progress-position': `${scrollProgress * 100}%`, '--brand-tilt': `${scrollProgress * 7}deg` }"
  >
    <nav class="nav shell-width" aria-label="主导航">
      <RouterLink class="brand" to="/" @click="menuOpen = false">
        <img :src="logoUrl" alt="" />
        <span>QZ Music</span>
        <i>v2</i>
      </RouterLink>

      <button class="icon-button mobile-menu" type="button" aria-label="打开导航" @click="menuOpen = !menuOpen">
        <X v-if="menuOpen" :size="20" />
        <Menu v-else :size="20" />
      </button>

      <div class="nav-center" :class="{ 'nav-center--open': menuOpen }">
        <RouterLink to="/" @click="menuOpen = false">首页</RouterLink>
        <RouterLink to="/blueprints" @click="menuOpen = false">
          蓝图
          <span class="nav-dot" />
        </RouterLink>
        <RouterLink to="/updates" @click="menuOpen = false">动态</RouterLink>
        <RouterLink to="/history" @click="menuOpen = false">更新</RouterLink>
        <RouterLink v-if="appStore.state.me?.isDeveloper" to="/admin/bans" @click="menuOpen = false">封禁管理</RouterLink>
        <RouterLink to="/docs/intro" :class="{ 'router-link-active': route.path.startsWith('/docs') }" @click="menuOpen = false">文档</RouterLink>
      </div>

      <div class="nav-actions">
        <button class="icon-button" type="button" aria-label="切换明暗主题" @click="appStore.toggleTheme()">
          <Sun v-if="appStore.state.theme === 'dark'" :size="18" />
          <Moon v-else :size="18" />
        </button>

        <div v-if="appStore.state.me" class="profile-wrap">
          <button class="profile-button" :class="{ 'profile-button--text-only': !appStore.state.me.picture }" type="button" @click="profileOpen = !profileOpen">
            <img v-if="appStore.state.me.picture" :src="appStore.state.me.picture" alt="" />
            <b>{{ appStore.state.me.name }}</b>
          </button>
          <Transition name="pop">
            <div v-if="profileOpen" class="profile-menu">
              <div>
                <strong>{{ appStore.state.me.name }}</strong>
                <small>{{ appStore.state.me.isDeveloper ? 'QZ 开发者' : '共创成员' }}</small>
              </div>
              <p>角色权限已从 Re-Link 同步</p>
              <button type="button" @click="logout"><LogOut :size="16" />退出登录</button>
            </div>
          </Transition>
        </div>
        <button v-else class="button button--compact" type="button" @click="appStore.login(route.fullPath)">
          <LogIn :size="17" /> 登录
        </button>
      </div>
    </nav>
    <span class="scroll-progress" aria-hidden="true" />
  </header>
</template>
