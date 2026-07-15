package main

import (
        "context"
        "database/sql"
        "encoding/json"
        "errors"
        "io"
        "net/http"
        "net/url"
        "os"
        "path/filepath"
        "strings"
        "time"
)

type Server struct {
        cfg   Config
        store *Store
        auth  *authService
}

type blueprintInput struct {
        Kind     string   `json:"kind"`
        Status   string   `json:"status"`
        Title    string   `json:"title"`
        Body     string   `json:"body"`
        Progress int      `json:"progress"`
        Images   []string `json:"images"`
}

type updateInput struct {
        Title string `json:"title"`
        Body  string `json:"body"`
        Scope string `json:"scope"`
}

func (s *Server) routes() http.Handler {
        mux := http.NewServeMux()
        mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, http.StatusOK, map[string]any{"ok": true}) })
        mux.HandleFunc("GET /api/me", s.handleMe)
        mux.HandleFunc("GET /api/blueprints", s.handleListBlueprints)
        mux.HandleFunc("POST /api/blueprints", s.handleCreateBlueprint)
        mux.HandleFunc("PATCH /api/blueprints/{id}", s.handleUpdateBlueprint)
        mux.HandleFunc("DELETE /api/blueprints/{id}", s.handleDeleteBlueprint)
        mux.HandleFunc("GET /api/updates", s.handleListUpdates)
        mux.HandleFunc("GET /api/history", s.handleHistory)
        mux.HandleFunc("POST /api/updates", s.handleCreateUpdate)
        mux.HandleFunc("PATCH /api/updates/{id}", s.handleUpdateUpdate)
        mux.HandleFunc("GET /api/{type}/{id}/comments", s.handleListComments)
        mux.HandleFunc("POST /api/{type}/{id}/comments", s.handleCreateComment)
        mux.HandleFunc("DELETE /api/{type}/{id}/comments/{commentId}", s.handleDeleteComment)
        mux.HandleFunc("POST /api/{type}/{id}/like", s.handleToggleLike)
        mux.HandleFunc("POST /api/blueprints/{id}/vote", s.handleVote)
        mux.HandleFunc("GET /api/admin/bans", s.handleListCommunityBans)
        mux.HandleFunc("DELETE /api/admin/bans/{userId}", s.handleRemoveCommunityBan)
        mux.HandleFunc("GET /api/config", s.handleGetSiteConfig)
        mux.HandleFunc("PUT /api/config", s.handleUpdateSiteConfig)
        mux.HandleFunc("GET /auth/login", s.auth.handleLogin)
        mux.HandleFunc("GET /auth/callback", s.auth.handleCallback)
        mux.HandleFunc("POST /auth/logout", s.auth.handleLogout)
        mux.Handle("/", spaHandler(s.cfg.WebDist))
        return s.recoverPanic(s.checkOrigin(s.auth.withUser(mux)))
}

func (s *Server) checkOrigin(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions {
                        origin := strings.TrimRight(r.Header.Get("Origin"), "/")
                        if origin != "" && origin != s.cfg.AllowedOrigin && !equivalentLoopbackOrigin(origin, s.cfg.AllowedOrigin) {
                                writeError(w, http.StatusForbidden, "origin_rejected", "请求来源不受信任")
                                return
                        }
                }
                next.ServeHTTP(w, r)
        })
}

func equivalentLoopbackOrigin(left, right string) bool {
        a, errA := url.Parse(left)
        b, errB := url.Parse(right)
        if errA != nil || errB != nil || a.Scheme != b.Scheme || a.Port() != b.Port() {
                return false
        }
        isLoopback := func(host string) bool { return host == "localhost" || host == "127.0.0.1" || host == "[::1]" }
        return isLoopback(a.Hostname()) && isLoopback(b.Hostname())
}

func (s *Server) recoverPanic(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                defer func() {
                        if recover() != nil {
                                writeError(w, http.StatusInternalServerError, "internal_error", "服务暂时遇到问题")
                        }
                }()
                next.ServeHTTP(w, r)
        })
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
        user := userFromContext(r.Context())
        userID := int64(0)
        if user != nil {
                userID = user.ID
        }
        limits, err := s.store.dailyLimits(r.Context(), userID)
        if err != nil {
                writeError(w, http.StatusInternalServerError, "limits_failed", "无法读取今日配额")
                return
        }
        banned := false
        if user != nil {
                banned, _ = s.store.isCommunityBanned(r.Context(), user.ID)
        }
        writeJSON(w, http.StatusOK, map[string]any{"user": user, "limits": limits, "communityBanned": banned})
}

func (s *Server) handleListBlueprints(w http.ResponseWriter, r *http.Request) {
        status, kind := r.URL.Query().Get("status"), r.URL.Query().Get("kind")
        if status != "" && !validStatus(status) {
                writeError(w, http.StatusBadRequest, "invalid_filter", "无效的蓝图状态")
                return
        }
        if kind != "" && kind != "preview" && kind != "request" {
                writeError(w, http.StatusBadRequest, "invalid_filter", "无效的蓝图类型")
                return
        }
        userID := contextUserID(r.Context())
        items, err := s.queryBlueprints(r.Context(), userID, status, kind)
        if err != nil {
                writeError(w, http.StatusInternalServerError, "query_failed", "无法读取蓝图")
                return
        }
        writeJSON(w, http.StatusOK, items)
}

func (s *Server) queryBlueprints(ctx context.Context, userID int64, status, kind string) ([]Blueprint, error) {
        query := `SELECT b.id, b.kind, b.status, b.title, b.body, b.progress,
                u.id, u.name, u.username, u.picture, u.is_developer,
                (SELECT COUNT(*) FROM likes l WHERE l.entity_type = 'blueprints' AND l.entity_id = b.id),
                (SELECT COUNT(*) FROM comments c WHERE c.entity_type = 'blueprints' AND c.entity_id = b.id),
                (SELECT COUNT(*) FROM votes v WHERE v.blueprint_id = b.id AND v.choice = 'want'),
                (SELECT COUNT(*) FROM votes v WHERE v.blueprint_id = b.id AND v.choice = 'dont_want'),
                EXISTS(SELECT 1 FROM likes l WHERE l.entity_type = 'blueprints' AND l.entity_id = b.id AND l.user_id = ?),
                COALESCE((SELECT v.choice FROM votes v WHERE v.blueprint_id = b.id AND v.user_id = ?), ''),
                b.created_at, b.updated_at
                FROM blueprints b JOIN users u ON u.id = b.author_id WHERE 1 = 1`
        args := []any{userID, userID}
        if status != "" {
                query += " AND b.status = ?"
                args = append(args, status)
        }
        if kind != "" {
                query += " AND b.kind = ?"
                args = append(args, kind)
        }
        query += ` ORDER BY CASE b.status WHEN 'in_progress' THEN 0 WHEN 'voting' THEN 1 WHEN 'request' THEN 2 WHEN 'released' THEN 3 ELSE 4 END, b.updated_at DESC LIMIT 100`
        rows, err := s.store.db.QueryContext(ctx, query, args...)
        if err != nil {
                return nil, err
        }
        defer rows.Close()
        items := make([]Blueprint, 0)
        for rows.Next() {
                var item Blueprint
                var developer, liked int
                var createdAt, updatedAt string
                if err := rows.Scan(&item.ID, &item.Kind, &item.Status, &item.Title, &item.Body, &item.Progress,
                        &item.Author.ID, &item.Author.Name, &item.Author.Username, &item.Author.Picture, &developer,
                        &item.LikeCount, &item.CommentCount, &item.Votes.Want, &item.Votes.DontWant, &liked, &item.Viewer.Vote, &createdAt, &updatedAt); err != nil {
                        return nil, err
                }
                item.Author.IsDeveloper = developer == 1
                item.Viewer.Liked = liked == 1
                item.CreatedAt = displayTime(createdAt, s.store.location)
                item.UpdatedAt = displayTime(updatedAt, s.store.location)
                item.Images, err = s.blueprintImages(ctx, item.ID)
                if err != nil {
                        return nil, err
                }
                items = append(items, item)
        }
        return items, rows.Err()
}

func (s *Server) blueprintImages(ctx context.Context, id int64) ([]string, error) {
        rows, err := s.store.db.QueryContext(ctx, `SELECT url FROM blueprint_images WHERE blueprint_id = ? ORDER BY position`, id)
        if err != nil {
                return nil, err
        }
        defer rows.Close()
        images := make([]string, 0)
        for rows.Next() {
                var value string
                if err := rows.Scan(&value); err != nil {
                        return nil, err
                }
                images = append(images, value)
        }
        return images, rows.Err()
}

func (s *Server) handleCreateBlueprint(w http.ResponseWriter, r *http.Request) {
        user := requireUser(w, r)
        if user == nil {
                return
        }
        if banned, err := s.store.isCommunityBanned(r.Context(), user.ID); err != nil {
                writeError(w, http.StatusInternalServerError, "ban_check_failed", "无法检查蓝图发布权限")
                return
        } else if banned {
                writeError(w, http.StatusForbidden, "community_banned", "你已被禁止发布功能请求和评论，请联系开发者")
                return
        }
        var input blueprintInput
        if !decodeJSON(w, r, &input, 64<<10) {
                return
        }
        if !user.IsDeveloper {
                input.Kind, input.Status, input.Progress = "request", "request", 0
        }
        if err := validateBlueprintInput(&input); err != nil {
                writeError(w, http.StatusBadRequest, "invalid_blueprint", err.Error())
                return
        }
        if input.Kind == "request" {
                limits, err := s.store.dailyLimits(r.Context(), user.ID)
                if err != nil {
                        writeError(w, http.StatusInternalServerError, "limits_failed", "无法读取今日配额")
                        return
                }
                if limits.RequestsRemaining <= 0 {
                        writeError(w, http.StatusTooManyRequests, "request_limit", "每位用户每天最多提交 2 个功能请求")
                        return
                }
        }
        item, err := s.insertBlueprint(r.Context(), *user, input)
        if errors.Is(err, errDailyLimit) {
                writeError(w, http.StatusTooManyRequests, "request_limit", "每位用户每天最多提交 2 个功能请求")
                return
        }
        if err != nil {
                writeError(w, http.StatusInternalServerError, "create_failed", "蓝图发布失败")
                return
        }
        limits, _ := s.store.dailyLimits(r.Context(), user.ID)
        writeJSON(w, http.StatusCreated, map[string]any{"item": item, "limits": limits})
}

func (s *Server) insertBlueprint(ctx context.Context, user User, input blueprintInput) (Blueprint, error) {
        tx, err := s.store.db.BeginTx(ctx, nil)
        if err != nil {
                return Blueprint{}, err
        }
        defer tx.Rollback()
        if input.Kind == "request" {
                if err := s.store.consumeDailyQuota(ctx, tx, user.ID, "requests"); err != nil {
                        return Blueprint{}, err
                }
        }
        now := time.Now().UTC().Format(time.RFC3339)
        result, err := tx.ExecContext(ctx, `INSERT INTO blueprints (kind, status, title, body, progress, author_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, input.Kind, input.Status, input.Title, input.Body, input.Progress, user.ID, now, now)
        if err != nil {
                return Blueprint{}, err
        }
        id, _ := result.LastInsertId()
        for index, image := range input.Images {
                if _, err := tx.ExecContext(ctx, `INSERT INTO blueprint_images (blueprint_id, position, url) VALUES (?, ?, ?)`, id, index, image); err != nil {
                        return Blueprint{}, err
                }
        }
        if err := tx.Commit(); err != nil {
                return Blueprint{}, err
        }
        return s.blueprintByID(ctx, id, user.ID)
}

func (s *Server) blueprintByID(ctx context.Context, id, userID int64) (Blueprint, error) {
        items, err := s.queryBlueprints(ctx, userID, "", "")
        if err != nil {
                return Blueprint{}, err
        }
        for _, item := range items {
                if item.ID == id {
                        return item, nil
                }
        }
        return Blueprint{}, sql.ErrNoRows
}

func (s *Server) handleUpdateBlueprint(w http.ResponseWriter, r *http.Request) {
        user := requireDeveloper(w, r)
        if user == nil {
                return
        }
        id, err := parseInt64(r.PathValue("id"))
        if err != nil {
                writeError(w, http.StatusBadRequest, "invalid_id", "无效的蓝图 ID")
                return
        }
        var input blueprintInput
        if !decodeJSON(w, r, &input, 64<<10) {
                return
        }
        if err := validateBlueprintInput(&input); err != nil {
                writeError(w, http.StatusBadRequest, "invalid_blueprint", err.Error())
                return
        }
        tx, err := s.store.db.BeginTx(r.Context(), nil)
        if err != nil {
                writeError(w, 500, "update_failed", "蓝图更新失败")
                return
        }
        defer tx.Rollback()
        result, err := tx.ExecContext(r.Context(), `UPDATE blueprints SET kind = ?, status = ?, title = ?, body = ?, progress = ?, updated_at = ? WHERE id = ?`, input.Kind, input.Status, input.Title, input.Body, input.Progress, time.Now().UTC().Format(time.RFC3339), id)
        if err != nil {
                writeError(w, 500, "update_failed", "蓝图更新失败")
                return
        }
        if affected, _ := result.RowsAffected(); affected == 0 {
                writeError(w, 404, "not_found", "蓝图不存在")
                return
        }
        if _, err := tx.ExecContext(r.Context(), `DELETE FROM blueprint_images WHERE blueprint_id = ?`, id); err != nil {
                writeError(w, 500, "update_failed", "蓝图更新失败")
                return
        }
        for index, image := range input.Images {
                if _, err := tx.ExecContext(r.Context(), `INSERT INTO blueprint_images (blueprint_id, position, url) VALUES (?, ?, ?)`, id, index, image); err != nil {
                        writeError(w, 500, "update_failed", "蓝图更新失败")
                        return
                }
        }
        if err := tx.Commit(); err != nil {
                writeError(w, 500, "update_failed", "蓝图更新失败")
                return
        }
        item, err := s.blueprintByID(r.Context(), id, user.ID)
        if err != nil {
                writeError(w, 500, "query_failed", "蓝图已更新但读取失败")
                return
        }
        writeJSON(w, http.StatusOK, map[string]any{"item": item})
}

func (s *Server) handleDeleteBlueprint(w http.ResponseWriter, r *http.Request) {
        developer := requireDeveloper(w, r)
        if developer == nil {
                return
        }
        id, err := parseInt64(r.PathValue("id"))
        if err != nil {
                writeError(w, http.StatusBadRequest, "invalid_id", "无效的蓝图 ID")
                return
        }
        var input struct {
                BanAuthor bool `json:"banAuthor"`
        }
        if r.ContentLength != 0 && !decodeJSON(w, r, &input, 2<<10) {
                return
        }
        tx, err := s.store.db.BeginTx(r.Context(), nil)
        if err != nil {
                writeError(w, http.StatusInternalServerError, "delete_failed", "蓝图删除失败")
                return
        }
        defer tx.Rollback()
        var authorID int64
        var authorIsDeveloper bool
        if err := tx.QueryRowContext(r.Context(), `SELECT u.id, u.is_developer FROM blueprints b JOIN users u ON u.id = b.author_id WHERE b.id = ?`, id).Scan(&authorID, &authorIsDeveloper); err != nil {
                if errors.Is(err, sql.ErrNoRows) {
                        writeError(w, http.StatusNotFound, "not_found", "蓝图不存在")
                } else {
                        writeError(w, http.StatusInternalServerError, "delete_failed", "蓝图删除失败")
                }
                return
        }
        if input.BanAuthor && authorIsDeveloper {
                writeError(w, http.StatusBadRequest, "developer_cannot_be_banned", "不能禁止开发者发布蓝图")
                return
        }
        now := time.Now().UTC().Format(time.RFC3339)
        if input.BanAuthor {
                if _, err := tx.ExecContext(r.Context(), `INSERT INTO community_bans (user_id, banned_by, created_at) VALUES (?, ?, ?) ON CONFLICT(user_id) DO UPDATE SET banned_by = excluded.banned_by, created_at = excluded.created_at`, authorID, developer.ID, now); err != nil {
                        writeError(w, http.StatusInternalServerError, "ban_failed", "蓝图删除成功前无法封禁用户")
                        return
                }
        }
        if _, err := tx.ExecContext(r.Context(), `DELETE FROM comments WHERE entity_type = 'blueprints' AND entity_id = ?`, id); err != nil {
                writeError(w, http.StatusInternalServerError, "delete_failed", "蓝图删除失败")
                return
        }
        if _, err := tx.ExecContext(r.Context(), `DELETE FROM likes WHERE entity_type = 'blueprints' AND entity_id = ?`, id); err != nil {
                writeError(w, http.StatusInternalServerError, "delete_failed", "蓝图删除失败")
                return
        }
        if _, err := tx.ExecContext(r.Context(), `DELETE FROM blueprints WHERE id = ?`, id); err != nil {
                writeError(w, http.StatusInternalServerError, "delete_failed", "蓝图删除失败")
                return
        }
        if err := tx.Commit(); err != nil {
                writeError(w, http.StatusInternalServerError, "delete_failed", "蓝图删除失败")
                return
        }
        writeJSON(w, http.StatusOK, map[string]any{"deleted": true, "banned": input.BanAuthor})
}

func (s *Server) handleListCommunityBans(w http.ResponseWriter, r *http.Request) {
        if requireDeveloper(w, r) == nil {
                return
        }
        rows, err := s.store.db.QueryContext(r.Context(), `SELECT
                u.id, u.name, u.username, u.picture, u.is_developer,
                actor.id, actor.name, actor.username, actor.picture, actor.is_developer,
                b.created_at
                FROM community_bans b
                JOIN users u ON u.id = b.user_id
                JOIN users actor ON actor.id = b.banned_by
                ORDER BY b.created_at DESC`)
        if err != nil {
                writeError(w, http.StatusInternalServerError, "query_failed", "无法读取封禁列表")
                return
        }
        defer rows.Close()
        items := make([]CommunityBan, 0)
        for rows.Next() {
                var item CommunityBan
                var createdAt string
                if err := rows.Scan(
                        &item.User.ID, &item.User.Name, &item.User.Username, &item.User.Picture, &item.User.IsDeveloper,
                        &item.BannedBy.ID, &item.BannedBy.Name, &item.BannedBy.Username, &item.BannedBy.Picture, &item.BannedBy.IsDeveloper,
                        &createdAt,
                ); err != nil {
                        writeError(w, http.StatusInternalServerError, "query_failed", "无法读取封禁列表")
                        return
                }
                item.CreatedAt = displayTime(createdAt, s.store.location)
                items = append(items, item)
        }
        if err := rows.Err(); err != nil {
                writeError(w, http.StatusInternalServerError, "query_failed", "无法读取封禁列表")
                return
        }
        writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleRemoveCommunityBan(w http.ResponseWriter, r *http.Request) {
        if requireDeveloper(w, r) == nil {
                return
        }
        userID, err := parseInt64(r.PathValue("userId"))
        if err != nil {
                writeError(w, http.StatusBadRequest, "invalid_id", "无效的用户 ID")
                return
        }
        result, err := s.store.db.ExecContext(r.Context(), `DELETE FROM community_bans WHERE user_id = ?`, userID)
        if err != nil {
                writeError(w, http.StatusInternalServerError, "unban_failed", "解除封禁失败")
                return
        }
        if affected, _ := result.RowsAffected(); affected == 0 {
                writeError(w, http.StatusNotFound, "not_found", "该用户不在封禁列表中")
                return
        }
        writeJSON(w, http.StatusOK, map[string]any{"removed": true})
}

func validateBlueprintInput(input *blueprintInput) error {
        var err error
        if input.Title, err = trimRequired(input.Title, 120); err != nil {
                return err
        }
        if input.Body, err = trimRequired(input.Body, 5000); err != nil {
                return err
        }
        if input.Kind != "preview" && input.Kind != "request" {
                return errors.New("无效的蓝图类型")
        }
        if !validStatus(input.Status) {
                return errors.New("无效的蓝图状态")
        }
        if input.Progress < 0 || input.Progress > 100 {
                return errors.New("进度必须在 0 到 100 之间")
        }
        if len(input.Images) > 5 {
                return errors.New("蓝图最多上传 5 张图片")
        }
        for _, image := range input.Images {
                if !validPublicImageURL(image) {
                        return errors.New("配图地址无效")
                }
        }
        return nil
}

func validStatus(value string) bool {
        switch value {
        case "request", "in_progress", "voting", "deprecated", "released":
                return true
        }
        return false
}

func validPublicImageURL(value string) bool {
        parsed, err := url.Parse(value)
        return err == nil && parsed.Scheme == "https" && parsed.Host != "" && len(value) <= 2048
}

func (s *Server) handleListUpdates(w http.ResponseWriter, r *http.Request) {
        items, err := s.queryUpdates(r.Context(), contextUserID(r.Context()))
        if err != nil {
                writeError(w, 500, "query_failed", "无法读取开发动态")
                return
        }
        writeJSON(w, 200, items)
}

func (s *Server) queryUpdates(ctx context.Context, userID int64) ([]Update, error) {
        rows, err := s.store.db.QueryContext(ctx, `SELECT p.id, p.title, p.body, p.scope, u.id, u.name, u.username, u.picture, u.is_developer,
                (SELECT COUNT(*) FROM likes l WHERE l.entity_type = 'updates' AND l.entity_id = p.id),
                (SELECT COUNT(*) FROM comments c WHERE c.entity_type = 'updates' AND c.entity_id = p.id),
                EXISTS(SELECT 1 FROM likes l WHERE l.entity_type = 'updates' AND l.entity_id = p.id AND l.user_id = ?), p.created_at, p.updated_at
                FROM updates p JOIN users u ON u.id = p.author_id ORDER BY p.created_at DESC LIMIT 100`, userID)
        if err != nil {
                return nil, err
        }
        defer rows.Close()
        items := make([]Update, 0)
        for rows.Next() {
                var item Update
                var developer, liked int
                var createdAt, updatedAt string
                if err := rows.Scan(&item.ID, &item.Title, &item.Body, &item.Scope, &item.Author.ID, &item.Author.Name, &item.Author.Username, &item.Author.Picture, &developer, &item.LikeCount, &item.CommentCount, &liked, &createdAt, &updatedAt); err != nil {
                        return nil, err
                }
                item.Author.IsDeveloper, item.Viewer.Liked = developer == 1, liked == 1
                item.CreatedAt, item.UpdatedAt = displayTime(createdAt, s.store.location), displayTime(updatedAt, s.store.location)
                items = append(items, item)
        }
        return items, rows.Err()
}

func (s *Server) handleCreateUpdate(w http.ResponseWriter, r *http.Request) {
        user := requireDeveloper(w, r)
        if user == nil {
                return
        }
        var input updateInput
        if !decodeJSON(w, r, &input, 128<<10) {
                return
        }
        var err error
        if input.Title, err = trimRequired(input.Title, 120); err != nil {
                writeError(w, 400, "invalid_update", err.Error())
                return
        }
        if input.Body, err = trimRequired(input.Body, 20000); err != nil {
                writeError(w, 400, "invalid_update", err.Error())
                return
        }
        now := time.Now().UTC().Format(time.RFC3339)
        result, err := s.store.db.ExecContext(r.Context(), `INSERT INTO updates (title, body, scope, author_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`, input.Title, input.Body, normalizeScope(input.Scope), user.ID, now, now)
        if err != nil {
                writeError(w, 500, "create_failed", "动态发布失败")
                return
        }
        id, _ := result.LastInsertId()
        items, _ := s.queryUpdates(r.Context(), user.ID)
        for _, item := range items {
                if item.ID == id {
                        writeJSON(w, 201, map[string]any{"item": item})
                        return
                }
        }
        writeError(w, 500, "query_failed", "动态已发布但读取失败")
}

func (s *Server) handleUpdateUpdate(w http.ResponseWriter, r *http.Request) {
        user := requireDeveloper(w, r)
        if user == nil {
                return
        }
        id, err := parseInt64(r.PathValue("id"))
        if err != nil {
                writeError(w, 400, "invalid_id", "无效的动态 ID")
                return
        }
        var input updateInput
        if !decodeJSON(w, r, &input, 128<<10) {
                return
        }
        if input.Title, err = trimRequired(input.Title, 120); err != nil {
                writeError(w, 400, "invalid_update", err.Error())
                return
        }
        if input.Body, err = trimRequired(input.Body, 20000); err != nil {
                writeError(w, 400, "invalid_update", err.Error())
                return
        }
        result, err := s.store.db.ExecContext(r.Context(), `UPDATE updates SET title = ?, body = ?, scope = ?, updated_at = ? WHERE id = ?`, input.Title, input.Body, normalizeScope(input.Scope), time.Now().UTC().Format(time.RFC3339), id)
        if err != nil {
                writeError(w, 500, "update_failed", "动态更新失败")
                return
        }
        if affected, _ := result.RowsAffected(); affected == 0 {
                writeError(w, 404, "not_found", "动态不存在")
                return
        }
        writeJSON(w, 200, map[string]any{"ok": true})
}

func (s *Server) handleListComments(w http.ResponseWriter, r *http.Request) {
        entityType := r.PathValue("type")
        id, err := parseInt64(r.PathValue("id"))
        if err != nil || !validEntityType(entityType) {
                writeError(w, 400, "invalid_target", "无效的评论对象")
                return
        }
        exists, err := s.store.entityExists(r.Context(), entityType, id)
        if err != nil {
                writeError(w, 500, "query_failed", "无法读取评论")
                return
        }
        if !exists {
                writeError(w, 404, "not_found", "内容不存在")
                return
        }
        rows, err := s.store.db.QueryContext(r.Context(), `SELECT c.id, c.body, c.created_at, u.id, u.name, u.username, u.picture, u.is_developer FROM comments c JOIN users u ON u.id = c.author_id WHERE c.entity_type = ? AND c.entity_id = ? ORDER BY c.created_at ASC LIMIT 300`, entityType, id)
        if err != nil {
                writeError(w, 500, "query_failed", "无法读取评论")
                return
        }
        defer rows.Close()
        comments := make([]Comment, 0)
        for rows.Next() {
                var item Comment
                var developer int
                var createdAt string
                if err := rows.Scan(&item.ID, &item.Body, &createdAt, &item.Author.ID, &item.Author.Name, &item.Author.Username, &item.Author.Picture, &developer); err != nil {
                        writeError(w, 500, "query_failed", "无法读取评论")
                        return
                }
                item.Author.IsDeveloper = developer == 1
                item.CreatedAt = displayTime(createdAt, s.store.location)
                comments = append(comments, item)
        }
        writeJSON(w, 200, comments)
}

func (s *Server) handleCreateComment(w http.ResponseWriter, r *http.Request) {
        user := requireUser(w, r)
        if user == nil {
                return
        }
        if banned, err := s.store.isCommunityBanned(r.Context(), user.ID); err != nil {
                writeError(w, http.StatusInternalServerError, "ban_check_failed", "无法检查评论权限")
                return
        } else if banned {
                writeError(w, http.StatusForbidden, "community_banned", "你已被禁止发布功能请求和评论，请联系开发者")
                return
        }
        entityType := r.PathValue("type")
        id, err := parseInt64(r.PathValue("id"))
        if err != nil || !validEntityType(entityType) {
                writeError(w, 400, "invalid_target", "无效的评论对象")
                return
        }
        exists, err := s.store.entityExists(r.Context(), entityType, id)
        if err != nil || !exists {
                writeError(w, 404, "not_found", "内容不存在")
                return
        }
        var input struct {
                Body string `json:"body"`
        }
        if !decodeJSON(w, r, &input, 4<<10) {
                return
        }
        input.Body, err = trimRequired(input.Body, 500)
        if err != nil {
                writeError(w, 400, "invalid_comment", err.Error())
                return
        }
        tx, err := s.store.db.BeginTx(r.Context(), nil)
        if err != nil {
                writeError(w, 500, "create_failed", "评论发布失败")
                return
        }
        defer tx.Rollback()
        if err := s.store.consumeDailyQuota(r.Context(), tx, user.ID, "comments"); errors.Is(err, errDailyLimit) {
                writeError(w, 429, "comment_limit", "每位用户每天最多发布 10 条评论")
                return
        } else if err != nil {
                writeError(w, 500, "create_failed", "评论发布失败")
                return
        }
        now := time.Now().UTC().Format(time.RFC3339)
        result, err := tx.ExecContext(r.Context(), `INSERT INTO comments (entity_type, entity_id, author_id, body, created_at) VALUES (?, ?, ?, ?, ?)`, entityType, id, user.ID, input.Body, now)
        if err != nil {
                writeError(w, 500, "create_failed", "评论发布失败")
                return
        }
        commentID, _ := result.LastInsertId()
        if err := tx.Commit(); err != nil {
                writeError(w, 500, "create_failed", "评论发布失败")
                return
        }
        limits, _ := s.store.dailyLimits(r.Context(), user.ID)
        writeJSON(w, 201, map[string]any{"comment": Comment{ID: commentID, Body: input.Body, Author: user.Public(), CreatedAt: displayTime(now, s.store.location)}, "limits": limits})
}

func (s *Server) handleDeleteComment(w http.ResponseWriter, r *http.Request) {
        developer := requireDeveloper(w, r)
        if developer == nil {
                return
        }
        entityType := r.PathValue("type")
        entityID, entityErr := parseInt64(r.PathValue("id"))
        commentID, commentErr := parseInt64(r.PathValue("commentId"))
        if entityErr != nil || commentErr != nil || !validEntityType(entityType) {
                writeError(w, http.StatusBadRequest, "invalid_target", "无效的评论对象")
                return
        }
        var input struct {
                BanAuthor bool `json:"banAuthor"`
        }
        if r.ContentLength != 0 && !decodeJSON(w, r, &input, 2<<10) {
                return
        }
        tx, err := s.store.db.BeginTx(r.Context(), nil)
        if err != nil {
                writeError(w, http.StatusInternalServerError, "delete_failed", "评论删除失败")
                return
        }
        defer tx.Rollback()
        var authorID int64
        var authorIsDeveloper bool
        if err := tx.QueryRowContext(r.Context(), `SELECT u.id, u.is_developer FROM comments c JOIN users u ON u.id = c.author_id WHERE c.id = ? AND c.entity_type = ? AND c.entity_id = ?`, commentID, entityType, entityID).Scan(&authorID, &authorIsDeveloper); err != nil {
                if errors.Is(err, sql.ErrNoRows) {
                        writeError(w, http.StatusNotFound, "not_found", "评论不存在")
                } else {
                        writeError(w, http.StatusInternalServerError, "delete_failed", "评论删除失败")
                }
                return
        }
        if input.BanAuthor && authorIsDeveloper {
                writeError(w, http.StatusBadRequest, "developer_cannot_be_banned", "不能封禁开发者")
                return
        }
        if input.BanAuthor {
                now := time.Now().UTC().Format(time.RFC3339)
                if _, err := tx.ExecContext(r.Context(), `INSERT INTO community_bans (user_id, banned_by, created_at) VALUES (?, ?, ?) ON CONFLICT(user_id) DO UPDATE SET banned_by = excluded.banned_by, created_at = excluded.created_at`, authorID, developer.ID, now); err != nil {
                        writeError(w, http.StatusInternalServerError, "ban_failed", "评论删除成功前无法封禁用户")
                        return
                }
        }
        if _, err := tx.ExecContext(r.Context(), `DELETE FROM comments WHERE id = ?`, commentID); err != nil {
                writeError(w, http.StatusInternalServerError, "delete_failed", "评论删除失败")
                return
        }
        if err := tx.Commit(); err != nil {
                writeError(w, http.StatusInternalServerError, "delete_failed", "评论删除失败")
                return
        }
        writeJSON(w, http.StatusOK, map[string]any{"deleted": true, "banned": input.BanAuthor})
}

func (s *Server) handleToggleLike(w http.ResponseWriter, r *http.Request) {
        user := requireUser(w, r)
        if user == nil {
                return
        }
        entityType := r.PathValue("type")
        id, err := parseInt64(r.PathValue("id"))
        if err != nil || !validEntityType(entityType) {
                writeError(w, 400, "invalid_target", "无效的点赞对象")
                return
        }
        exists, err := s.store.entityExists(r.Context(), entityType, id)
        if err != nil || !exists {
                writeError(w, 404, "not_found", "内容不存在")
                return
        }
        tx, err := s.store.db.BeginTx(r.Context(), nil)
        if err != nil {
                writeError(w, 500, "like_failed", "点赞失败")
                return
        }
        defer tx.Rollback()
        result, err := tx.ExecContext(r.Context(), `DELETE FROM likes WHERE entity_type = ? AND entity_id = ? AND user_id = ?`, entityType, id, user.ID)
        if err != nil {
                writeError(w, 500, "like_failed", "点赞失败")
                return
        }
        affected, _ := result.RowsAffected()
        liked := affected == 0
        if liked {
                if _, err := tx.ExecContext(r.Context(), `INSERT INTO likes (entity_type, entity_id, user_id, created_at) VALUES (?, ?, ?, ?)`, entityType, id, user.ID, time.Now().UTC().Format(time.RFC3339)); err != nil {
                        writeError(w, 500, "like_failed", "点赞失败")
                        return
                }
        }
        var count int
        if err := tx.QueryRowContext(r.Context(), `SELECT COUNT(*) FROM likes WHERE entity_type = ? AND entity_id = ?`, entityType, id).Scan(&count); err != nil {
                writeError(w, 500, "like_failed", "点赞失败")
                return
        }
        if err := tx.Commit(); err != nil {
                writeError(w, 500, "like_failed", "点赞失败")
                return
        }
        writeJSON(w, 200, map[string]any{"liked": liked, "count": count})
}

func (s *Server) handleVote(w http.ResponseWriter, r *http.Request) {
        user := requireUser(w, r)
        if user == nil {
                return
        }
        id, err := parseInt64(r.PathValue("id"))
        if err != nil {
                writeError(w, 400, "invalid_id", "无效的蓝图 ID")
                return
        }
        var input struct {
                Choice string `json:"choice"`
        }
        if !decodeJSON(w, r, &input, 2<<10) {
                return
        }
        if input.Choice != "want" && input.Choice != "dont_want" {
                writeError(w, 400, "invalid_vote", "无效的投票选项")
                return
        }
        var status string
        if err := s.store.db.QueryRowContext(r.Context(), `SELECT status FROM blueprints WHERE id = ?`, id).Scan(&status); err != nil {
                writeError(w, 404, "not_found", "蓝图不存在")
                return
        }
        if status != "voting" {
                writeError(w, 409, "not_voting", "这份蓝图当前不在投票阶段")
                return
        }
        now := time.Now().UTC().Format(time.RFC3339)
        _, err = s.store.db.ExecContext(r.Context(), `INSERT INTO votes (blueprint_id, user_id, choice, created_at, updated_at) VALUES (?, ?, ?, ?, ?) ON CONFLICT(blueprint_id, user_id) DO UPDATE SET choice = excluded.choice, updated_at = excluded.updated_at`, id, user.ID, input.Choice, now, now)
        if err != nil {
                writeError(w, 500, "vote_failed", "投票失败")
                return
        }
        var votes Votes
        _ = s.store.db.QueryRowContext(r.Context(), `SELECT SUM(CASE WHEN choice = 'want' THEN 1 ELSE 0 END), SUM(CASE WHEN choice = 'dont_want' THEN 1 ELSE 0 END) FROM votes WHERE blueprint_id = ?`, id).Scan(&votes.Want, &votes.DontWant)
        writeJSON(w, 200, map[string]any{"choice": input.Choice, "votes": votes})
}

func requireUser(w http.ResponseWriter, r *http.Request) *User {
        user := userFromContext(r.Context())
        if user == nil {
                writeError(w, 401, "authentication_required", "登录后才能进行此操作")
                return nil
        }
        return user
}

func requireDeveloper(w http.ResponseWriter, r *http.Request) *User {
        user := requireUser(w, r)
        if user == nil {
                return nil
        }
        if !user.IsDeveloper {
                writeError(w, 403, "developer_required", "只有开发者可以进行此操作")
                return nil
        }
        return user
}

func contextUserID(ctx context.Context) int64 {
        if user := userFromContext(ctx); user != nil {
                return user.ID
        }
        return 0
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any, maxBytes int64) bool {
        r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
        decoder := json.NewDecoder(r.Body)
        decoder.DisallowUnknownFields()
        if err := decoder.Decode(target); err != nil {
                writeError(w, 400, "invalid_json", "请求内容格式不正确")
                return false
        }
        if err := decoder.Decode(&struct{}{}); err != io.EOF {
                writeError(w, 400, "invalid_json", "请求只能包含一个 JSON 对象")
                return false
        }
        return true
}

func writeJSON(w http.ResponseWriter, status int, value any) {
        w.Header().Set("Content-Type", "application/json; charset=utf-8")
        w.Header().Set("Cache-Control", "no-store")
        w.WriteHeader(status)
        _ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
        writeJSON(w, status, map[string]any{"code": code, "message": message})
}

func spaHandler(dist string) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                if r.Method != http.MethodGet && r.Method != http.MethodHead {
                        http.NotFound(w, r)
                        return
                }
                cleanPath := filepath.Clean(strings.TrimPrefix(r.URL.Path, "/"))
                if cleanPath == "." {
                        cleanPath = "index.html"
                }
                candidate := filepath.Join(dist, cleanPath)
                if absolute, err := filepath.Abs(candidate); err == nil {
                        root, _ := filepath.Abs(dist)
                        if relative, err := filepath.Rel(root, absolute); err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
                                if stat, err := os.Stat(absolute); err == nil && !stat.IsDir() {
                                        http.ServeFile(w, r, absolute)
                                        return
                                }
                        }
                }
                index := filepath.Join(dist, "index.html")
                if stat, err := os.Stat(index); err == nil && !stat.IsDir() {
                        http.ServeFile(w, r, index)
                        return
                }
                http.NotFound(w, r)
        })
}

func (s *Server) handleGetSiteConfig(w http.ResponseWriter, r *http.Request) {
        rows, err := s.store.db.QueryContext(r.Context(), `SELECT key, value FROM site_config`)
        if err != nil {
                writeError(w, http.StatusInternalServerError, "config_failed", "无法读取站点配置")
                return
        }
        defer rows.Close()
        config := map[string]string{}
        for rows.Next() {
                var key, value string
                if err := rows.Scan(&key, &value); err != nil {
                        continue
                }
                config[key] = value
        }
        writeJSON(w, http.StatusOK, config)
}

func (s *Server) handleUpdateSiteConfig(w http.ResponseWriter, r *http.Request) {
        user := requireDeveloper(w, r)
        if user == nil {
                return
        }
        var input map[string]string
        if !decodeJSON(w, r, &input, 256<<10) {
                return
        }
        if len(input) == 0 {
                writeError(w, http.StatusBadRequest, "empty_config", "配置内容不能为空")
                return
        }
        now := time.Now().UTC().Format(time.RFC3339)
        tx, err := s.store.db.BeginTx(r.Context(), nil)
        if err != nil {
                writeError(w, http.StatusInternalServerError, "config_failed", "配置保存失败")
                return
        }
        defer tx.Rollback()
        for key, value := range input {
                if strings.TrimSpace(key) == "" || len(key) > 64 {
                        continue
                }
                if len(value) > 10000 {
                        value = string([]rune(value)[:10000])
                }
                _, err := tx.ExecContext(r.Context(), `INSERT INTO site_config (key, value, updated_at, updated_by) VALUES (?, ?, ?, ?)
                        ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at, updated_by = excluded.updated_by`,
                        key, value, now, user.ID)
                if err != nil {
                        writeError(w, http.StatusInternalServerError, "config_failed", "配置保存失败")
                        return
                }
        }
        if err := tx.Commit(); err != nil {
                writeError(w, http.StatusInternalServerError, "config_failed", "配置保存失败")
                return
        }
        writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
