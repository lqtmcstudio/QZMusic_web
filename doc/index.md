# QZ Music v2 项目介绍

**更新日期：** `2025.12.17`

## 📖 软件概览

* **中文全称：** 清泽音乐
* **英文名称：** QZ Music
* **项目核心：** 致力于打造极致纯净、功能强大的现代化音乐播放器。

---

## 🚀 版本迭代 (重要区分)

QZ Music 经历了从底层架构到 UI 设计的彻底重构，请务必区分 v1 与 v2 版本：

| 特性       | v1.x.x (旧版)                                                 | v2.x.x (新版)                                         |
|----------|-------------------------------------------------------------|-----------------------------------------------------|
| **开发语言** | Flutter (Dart)                                              | **Jetpack Compose (Kotlin)**                        |
| **插件系统** | QuickJS                                                     | **NodeJS**                                          |
| **维护状态** | 已停止维护 (已开源)                                                 | **持续迭代中 (半开源)**                                     |
| **软件特点** | 界面朴素，播放逻辑存留少量问题                                             | 现代化布局，响应式交互，高度自定义                                   |
| **开源部分** | [GitHub 仓库](https://github.com/lqtmcstudio/QZMusic_Flutter) | [Github 仓库](https://github.com/lqtmcstudio/QZMusic) |

> **⚠️ Tip：** v1 与 v2 之间的代码、界面、逻辑以及插件系统**完全不通用**。  
> **⚠️ Tip2：** v2**正在开发中**,因学业紧张所以开发效率和进度不定,具体进度见官网/群内通知。   

---

## ⚖️ 开源协议与组件说明

本项目严格遵守第三方开源协议

### 使用的开源组件

* **组件名称：** [Accompanist (lyrics-ui / lyrics-core)](https://github.com/6xingyv/accompanist-lyrics-ui)
* **组件用途：** KMP 跨平台歌词显示组件
* **原作者：** 阿睿睡不够
* **授权协议：** [Apache License 2.0](http://www.apache.org/licenses/LICENSE-2.0.txt)
* **修改部分：** 手动兼容安卓Jetpack Compose/加入弹簧滚动/优化滚动 (所有修改后的代码仅项目内使用,未进行二次代码公开/分享)

### 合规性声明

1. **独立性：** 本项目**未涉及**任何使用 GPL/AGPL 协议的组件（如 AMLL 等），因此根据现行协议，本项目无需强制完整开源。
2. **维权与反馈：** 我们极力维护开源生态的健康。若您认为本项目违反了任何开源协议或侵犯了您的权益，请通过以下方式联系：
* **私信联系：** 软件制作者
* **官方邮箱：** [lqtmcstudio@gmail.com](mailto:lqtmcstudio@gmail.com)


3. **详细条款：** 更多法律声明与隐私协议请参阅：[QZ Music 法务声明](https://music.qz.shiqianjiang.cn/guide/law.html)

---

## 🛠️ 开发团队

* **蜻蜓T-T (lqtmcstudio)**：核心架构 / 插件系统 / 逻辑开发 / UI开发
* **rvntd (nevodev)**：UI/UX 设计/开发 / 交互优化 / 代码结构优化