<script setup>
import { ArrowRight, AudioLines, Blocks, Cloud, Code2, Gauge, Github, Sparkles, Tv } from 'lucide-vue-next'
import { RouterLink } from 'vue-router'
import logoUrl from '../../src/icon.png'

const features = [
  { icon: Blocks, title: '插件化', copy: '基于 NodeJS 的插件体系，让音乐能力自由延伸。' },
  { icon: AudioLines, title: '纯粹播放', copy: '专注播放、歌词与手感，不堆砌臃肿功能。' },
  { icon: Cloud, title: '云端同步', copy: '同步歌单与偏好设置，不存储你的音频文件。' },
  { icon: Gauge, title: 'Compose UI', copy: 'Jetpack Compose 构建的新一代流畅交互。' },
]

const contributors = [
  {
    name: 'miao',
    handle: '@miao-moe',
    avatar: 'https://avatars.githubusercontent.com/u/108458240?v=4',
    github: 'https://github.com/miao-moe',
    bilibili: 'https://space.bilibili.com/3546566025140405',
    tags: [
      { icon: Code2, label: '官网重构' },
      { icon: Tv, label: 'PC 端维护' },
    ],
  },
]
</script>

<template>
  <div class="home-view">
    <section class="home-hero shell-width">
      <div class="hero-copy">
        <div class="hero-badge"><span /> QZ Music v2 正在持续生长</div>
        <h1>让每一次播放，<br /><em>都有回响。</em></h1>
        <p>一款纯净、流畅，也认真倾听每种想法的多功能音乐播放器。</p>
        <div class="hero-actions">
          <RouterLink class="button" to="/blueprints">走进蓝图 <ArrowRight :size="18" /></RouterLink>
          <RouterLink class="text-link" to="/updates">阅读开发动态 <span>↗</span></RouterLink>
        </div>
        <div class="hero-note"><Sparkles :size="17" /> 你提出的下一个想法，或许正是我们正在打磨的功能。</div>
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
      <p>从 Flutter 到 Jetpack Compose，从单一脚本到 NodeJS 插件系统，QZ Music 的每一次重构，都为了更轻、更自由。如今，开发过程也向你敞开。</p>
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
          <p>查看实时进度、提交功能请求，为真正想要的功能投上一票。</p>
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

    <section class="contributors shell-width">
      <div class="section-heading">
        <span class="eyebrow">BUILT TOGETHER</span>
        <h2>那些让 QZ Music<br />变得更好的人。</h2>
      </div>
      <div class="contributor-grid">
        <article v-for="person in contributors" :key="person.handle" class="contributor-card">
          <header>
            <img :src="person.avatar" :alt="person.name" loading="lazy" />
            <div>
              <strong>{{ person.name }}</strong>
              <a :href="person.github" target="_blank" rel="noreferrer noopener">{{ person.handle }}</a>
            </div>
          </header>
          <ul class="contributor-tags">
            <li v-for="tag in person.tags" :key="tag.label">
              <component :is="tag.icon" :size="14" /> {{ tag.label }}
            </li>
          </ul>
          <footer>
            <a class="contributor-link contributor-link--github" :href="person.github" target="_blank" rel="noreferrer noopener">
              <Github :size="16" /> GitHub
            </a>
            <a v-if="person.bilibili" class="contributor-link contributor-link--bili" :href="person.bilibili" target="_blank" rel="noreferrer noopener">
              <Tv :size="16" /> 哔哩哔哩
            </a>
          </footer>
        </article>
      </div>
    </section>
  </div>
</template>
