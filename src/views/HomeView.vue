<script setup>
import { ArrowRight, AudioLines, Blocks, Cloud, Code2, Gauge, Sparkles } from 'lucide-vue-next'
import { RouterLink } from 'vue-router'
import logoUrl from '../../old/assets/logo.png'

const features = [
  { icon: Blocks, title: '插件化', copy: '以 NodeJS 为插件基础平台，让音乐能力自由延伸。' },
  { icon: AudioLines, title: '纯粹播放', copy: '专注播放、歌词与手感，不把体验交给臃肿功能。' },
  { icon: Cloud, title: '云端同步', copy: '同步歌单配置与偏好，不存储你的音频文件。' },
  { icon: Gauge, title: 'Compose UI', copy: 'Jetpack Compose 构建的新一代流畅交互。' },
]
</script>

<template>
  <div class="home-view">
    <section class="home-hero shell-width">
      <div class="hero-copy">
        <div class="hero-badge"><span /> QZ Music v2 正在持续生长</div>
        <h1>让每一次播放，<br /><em>都有回响。</em></h1>
        <p>一款纯净、流畅，也愿意认真听取每个想法的多功能音乐播放器。</p>
        <div class="hero-actions">
          <RouterLink class="button" to="/blueprints">走进蓝图 <ArrowRight :size="18" /></RouterLink>
          <RouterLink class="text-link" to="/updates">阅读开发动态 <span>↗</span></RouterLink>
        </div>
        <div class="hero-note"><Sparkles :size="17" /> 你提出的下一个想法，可能就是我们正在制作的功能。</div>
      </div>

      <div class="hero-visual" aria-label="QZ Music 产品视觉">
        <div class="orb orb--one" />
        <div class="orb orb--two" />
        <div class="logo-stage">
          <img :src="logoUrl" alt="QZ Music 音符图标" />
        </div>
        <div class="floating-card floating-card--progress">
          <span><i /> NOW BUILDING</span>
          <strong>播放页动效重构</strong>
          <div><i style="width: 72%" /></div>
          <small>72% · 本周 +8%</small>
        </div>
        <div class="floating-card floating-card--signal">
          <div class="signal-bars"><i /><i /><i /><i /><i /></div>
          <span>Hi-Res</span>
          <small>纯净播放中</small>
        </div>
      </div>
    </section>

    <section class="home-manifesto shell-width">
      <div class="section-heading">
        <span class="eyebrow">BUILT WITH THE COMMUNITY</span>
        <h2>播放器由代码构成，<br />体验由每个人共同完成。</h2>
      </div>
      <p>从 Flutter 到 Jetpack Compose，从单一脚本到 NodeJS 插件系统，QZ Music 的每次重构都为了更轻、更自由。现在，开发过程也向你打开。</p>
    </section>

    <section class="feature-grid shell-width">
      <article v-for="feature in features" :key="feature.title">
        <component :is="feature.icon" :size="24" />
        <h3>{{ feature.title }}</h3>
        <p>{{ feature.copy }}</p>
      </article>
    </section>

    <section class="portal-grid shell-width">
      <RouterLink class="portal portal--blueprint" to="/blueprints">
        <div>
          <span class="eyebrow">BLUEPRINT</span>
          <h2>下一曲，<br />由你参与。</h2>
          <p>查看实时进度、提交功能请求，为真正想要的功能投票。</p>
        </div>
        <div class="portal-icon"><Code2 :size="38" /></div>
        <span class="portal-arrow">↗</span>
      </RouterLink>
      <RouterLink class="portal portal--updates" to="/updates">
        <div>
          <span class="eyebrow">DEV LOG</span>
          <h2>不只告诉你<br />“更新了什么”。</h2>
          <p>记录功能背后的取舍、细节，以及一段代码如何成为体验。</p>
        </div>
        <div class="portal-icon"><AudioLines :size="38" /></div>
        <span class="portal-arrow">↗</span>
      </RouterLink>
    </section>
  </div>
</template>
