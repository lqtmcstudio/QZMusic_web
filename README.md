# QZ Music 官方网站

使用 Vue 3 + Go 重构的 QZ Music 官网。前端包含首页、蓝图广场、开发动态广场和项目更新历史；Go 服务负责 Re-Link SSO、权限、持久化以及全部互动规则，图片由浏览器直接上传。

## 已实现

- 蓝图：开发预告、实时进度、用户功能请求、开发者审阅/修改，以及制作中、待投票、已废弃、已完成状态。
- 互动：登录用户可点赞、评论；待投票蓝图支持“想要 / 不想要”，同一用户只保留一个当前选择。
- 限额：每个用户每天最多 10 条评论、2 个功能请求。计数在 SQLite 事务中原子执行，不能只靠前端绕过。
- 配图：蓝图最多 5 张；动态编辑器可上传图片并自动插入 Markdown。
- 动态：仅开发者可发布和编辑，所有访客可阅读，登录用户可评论和点赞。
- 更新历史：分别缓存 Android 与 Windows 仓库 `master` 分支的 Commit，展示提交者、时间、文件数及代码增减量。
- 社区管理：开发者可删除蓝图，或右键删除评论，并可同时封禁作者；统一封禁会暂停该用户的功能请求与评论权限，开发者可从导航栏进入封禁管理并解除。
- 认证：Re-Link OAuth 2.0 Authorization Code + PKCE S256；服务端交换授权码，校验 RS256、issuer、audience、过期时间和 nonce。
- 会话：浏览器只保存 HttpOnly、SameSite=Lax 的本地 Session Cookie，不暴露 OAuth Token。

## 本地开发

1. 在 Re-Link 开发者中心创建 OAuth Client，回调地址登记为：

   ```text
   http://localhost:8787/auth/callback
   ```

2. 复制 `backend/.env.example` 为 `backend/.env`，填写：

   ```dotenv
   OAUTH_CLIENT_ID=linkc_xxx
   OAUTH_CLIENT_SECRET=links_xxx
   GITHUB_API_KEY=github_pat_xxx
   ```

3. 启动 Go 服务：

   ```powershell
   cd backend
   go run .
   ```

4. 另开终端启动 Vue：

   ```powershell
   npm install
   npm run dev
   ```

访问 `http://localhost:5173`。未配置 SSO 时，广场仍可匿名浏览，但所有写操作会要求登录。

## 配置开发者

本站从 Re-Link UserInfo 的 `roles` 数组读取权限，并用稳定的 `roles[].id` 判断开发者。把开发者角色 ID 写入 `backend/.env`：

```dotenv
DEVELOPER_ROLE_ID=developer
```

每次 OAuth 登录都会重新读取 UserInfo 并覆盖本地 `is_developer`，因此 Re-Link 侧增删角色后，用户重新登录即可同步。角色名称、邮箱、用户名和 `sub` 都不会被用作开发者授权依据。

## 生产运行

先构建前端：

```powershell
npm run build
```

随后从 `backend` 目录启动 Go 服务。默认 `WEB_DIST=../dist`，Go 会同时提供 API 和 Vue 单页应用：

```powershell
cd backend
go run .
```

生产环境请使用 HTTPS，并把 `APP_URL`、`OAUTH_REDIRECT_URI`、`COOKIE_SECURE=true` 与 Re-Link 中登记的回调地址设为完全一致的正式域名。

## 数据与上传

- SQLite 默认位置：`backend/data/qz-music.db`。
- 数据库会在首次启动时自动建表，不会写入示例内容。
- GitHub Commit 会长期保存在 SQLite。首次同步最多回填两个仓库各 1000 条 `master` Commit，之后每 10 分钟读取最新 100 条并只增量写入。
- `GITHUB_API_KEY` 需要能够读取私有仓库 `nevodev/QZ-Music`；PC 仓库为 `lqtmcstudio/QZMusic_PC`。
- 图片由浏览器直接上传到 Supabase Edge Function，不再经过 Go 服务；单文件限制为 25MB。
- 前端上传地址和 anon key 可在项目根目录 `.env` 中通过 `VITE_SUPABASE_URL`、`VITE_SUPABASE_ANON_KEY` 覆盖。

## 校验

```powershell
npm run build
cd backend
go test ./...
go build ./...
```
