import { createApp } from 'vue'
import { createRouter, createWebHistory } from 'vue-router'
import App from './App.vue'
import HomeView from './views/HomeView.vue'
import BlueprintsView from './views/BlueprintsView.vue'
import UpdatesView from './views/UpdatesView.vue'
import DocsView from './views/DocsView.vue'
import HistoryView from './views/HistoryView.vue'
import BanManagementView from './views/BanManagementView.vue'
import './style.css'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', component: HomeView, meta: { title: 'QZ Music' } },
    { path: '/blueprints', component: BlueprintsView, meta: { title: '蓝图' } },
    { path: '/updates', component: UpdatesView, meta: { title: '动态' } },
    { path: '/history', component: HistoryView, meta: { title: '更新历史' } },
    { path: '/admin/bans', component: BanManagementView, meta: { title: '封禁管理' } },
    { path: '/docs/:pathMatch(.*)*', component: DocsView, meta: { title: '使用文档' } },
  ],
  scrollBehavior: () => ({ top: 0, behavior: 'smooth' }),
})

router.afterEach((to) => {
  document.title = `${to.meta.title} · QZ Music`
})

createApp(App).use(router).mount('#app')
