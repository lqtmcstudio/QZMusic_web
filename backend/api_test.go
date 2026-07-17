package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

type testApp struct {
	handler  http.Handler
	store    *Store
	auth     *authService
	user     User
	dev      User
	token    string
	devToken string
}

func newTestApp(t *testing.T) testApp {
	t.Helper()
	location, _ := time.LoadLocation("Asia/Shanghai")
	store, err := openStore(filepath.Join(t.TempDir(), "test.db"), location)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.db.Close() })
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := store.db.Exec(`INSERT INTO users (issuer, subject, name, username, email, picture, is_developer, created_at, last_seen_at) VALUES ('test', 'member', '测试成员', '', '', '', 0, ?, ?)`, now, now)
	if err != nil {
		t.Fatal(err)
	}
	userID, _ := result.LastInsertId()
	user := User{ID: userID, Issuer: "test", Subject: "member", Name: "测试成员"}
	devResult, err := store.db.Exec(`INSERT INTO users (issuer, subject, name, username, email, picture, is_developer, created_at, last_seen_at) VALUES ('test', 'developer', '测试开发者', '', '', '', 1, ?, ?)`, now, now)
	if err != nil {
		t.Fatal(err)
	}
	devID, _ := devResult.LastInsertId()
	dev := User{ID: devID, Issuer: "test", Subject: "developer", Name: "测试开发者", IsDeveloper: true}
	if _, err := store.db.Exec(`INSERT INTO blueprints (kind, status, title, body, progress, author_id, created_at, updated_at) VALUES ('request', 'request', '测试蓝图', '测试用蓝图内容', 0, ?, ?, ?)`, dev.ID, now, now); err != nil {
		t.Fatal(err)
	}
	token, devToken := "member-session", "developer-session"
	expires := time.Now().UTC().Add(time.Hour).Format(time.RFC3339)
	if _, err := store.db.Exec(`INSERT INTO sessions (token_hash, user_id, expires_at, created_at) VALUES (?, ?, ?, ?), (?, ?, ?, ?)`, hashToken(token), user.ID, expires, now, hashToken(devToken), dev.ID, expires, now); err != nil {
		t.Fatal(err)
	}
	cfg := Config{AppURL: "http://example.test", AllowedOrigin: "http://example.test", WebDist: t.TempDir(), SessionDuration: time.Hour, QuotaLocation: location, DeveloperRoleID: "developer"}
	auth := newAuthService(cfg, store)
	server := &Server{cfg: cfg, store: store, auth: auth}
	return testApp{handler: server.routes(), store: store, auth: auth, user: user, dev: dev, token: token, devToken: devToken}
}

func (a testApp) request(t *testing.T, method, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var payload bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&payload).Encode(body); err != nil {
			t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, path, &payload)
	req.Header.Set("Origin", "http://example.test")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.AddCookie(&http.Cookie{Name: "qz_session", Value: token})
	}
	recorder := httptest.NewRecorder()
	a.handler.ServeHTTP(recorder, req)
	return recorder
}

func TestPublicReadAndProtectedWrite(t *testing.T) {
	app := newTestApp(t)
	if response := app.request(t, http.MethodGet, "/api/blueprints", "", nil); response.Code != http.StatusOK {
		t.Fatalf("public list status = %d", response.Code)
	}
	input := blueprintInput{Kind: "request", Status: "request", Title: "匿名请求", Body: "不应被创建"}
	if response := app.request(t, http.MethodPost, "/api/blueprints", "", input); response.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous write status = %d", response.Code)
	}
}

func TestDailyLimitsAreEnforced(t *testing.T) {
	app := newTestApp(t)
	for index := 0; index < 3; index++ {
		response := app.request(t, http.MethodPost, "/api/blueprints", app.token, blueprintInput{Kind: "preview", Status: "released", Title: "功能请求", Body: "希望增加一个更好用的能力"})
		expected := http.StatusCreated
		if index == 2 {
			expected = http.StatusTooManyRequests
		}
		if response.Code != expected {
			t.Fatalf("request %d status = %d, want %d: %s", index+1, response.Code, expected, response.Body.String())
		}
	}
	for index := 0; index < 11; index++ {
		response := app.request(t, http.MethodPost, "/api/blueprints/1/comments", app.token, map[string]string{"body": "认真反馈一下这个功能"})
		expected := http.StatusCreated
		if index == 10 {
			expected = http.StatusTooManyRequests
		}
		if response.Code != expected {
			t.Fatalf("comment %d status = %d, want %d: %s", index+1, response.Code, expected, response.Body.String())
		}
	}
}

func TestDeveloperReviewAndSingleVote(t *testing.T) {
	app := newTestApp(t)
	input := blueprintInput{Kind: "request", Status: "voting", Title: "桌面歌词投票", Body: "请大家决定是否优先制作", Progress: 0, Images: []string{}}
	response := app.request(t, http.MethodPatch, "/api/blueprints/1", app.devToken, input)
	if response.Code != http.StatusOK {
		t.Fatalf("developer review status = %d: %s", response.Code, response.Body.String())
	}
	response = app.request(t, http.MethodPost, "/api/blueprints/1/vote", app.token, map[string]string{"choice": "want"})
	if response.Code != http.StatusOK {
		t.Fatalf("first vote status = %d: %s", response.Code, response.Body.String())
	}
	response = app.request(t, http.MethodPost, "/api/blueprints/1/vote", app.token, map[string]string{"choice": "dont_want"})
	if response.Code != http.StatusOK {
		t.Fatalf("changed vote status = %d: %s", response.Code, response.Body.String())
	}
	var result struct {
		Votes Votes `json:"votes"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Votes.Want != 0 || result.Votes.DontWant != 1 {
		t.Fatalf("vote counts = %+v", result.Votes)
	}

	response = app.request(t, http.MethodPost, "/api/updates", app.token, updateInput{Title: "普通用户动态", Body: "不应发布"})
	if response.Code != http.StatusForbidden {
		t.Fatalf("member update status = %d", response.Code)
	}
	response = app.request(t, http.MethodPost, "/api/updates", app.devToken, updateInput{Title: "开发动态", Body: "开发者可以发布"})
	if response.Code != http.StatusCreated {
		t.Fatalf("developer update status = %d: %s", response.Code, response.Body.String())
	}
}

func TestDeveloperCanDeleteBanAndUnbanBlueprintAuthor(t *testing.T) {
	app := newTestApp(t)
	created := app.request(t, http.MethodPost, "/api/blueprints", app.token, blueprintInput{
		Kind: "request", Title: "希望增加均衡器", Body: "这是待开发者审阅的功能请求",
	})
	if created.Code != http.StatusCreated {
		t.Fatalf("create status = %d: %s", created.Code, created.Body.String())
	}
	var payload struct {
		Item Blueprint `json:"item"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	deleted := app.request(t, http.MethodDelete, "/api/blueprints/"+strconv.FormatInt(payload.Item.ID, 10), app.devToken, map[string]bool{"banAuthor": true})
	if deleted.Code != http.StatusOK {
		t.Fatalf("delete and ban status = %d: %s", deleted.Code, deleted.Body.String())
	}

	blocked := app.request(t, http.MethodPost, "/api/blueprints", app.token, blueprintInput{
		Kind: "request", Title: "被禁止后的请求", Body: "这条内容不应被创建",
	})
	if blocked.Code != http.StatusForbidden {
		t.Fatalf("blocked create status = %d: %s", blocked.Code, blocked.Body.String())
	}

	commentBlocked := app.request(t, http.MethodPost, "/api/blueprints/1/comments", app.token, map[string]string{"body": "封禁后也不能评论"})
	if commentBlocked.Code != http.StatusForbidden {
		t.Fatalf("blocked comment status = %d: %s", commentBlocked.Code, commentBlocked.Body.String())
	}

	bans := app.request(t, http.MethodGet, "/api/admin/bans", app.devToken, nil)
	if bans.Code != http.StatusOK {
		t.Fatalf("ban list status = %d: %s", bans.Code, bans.Body.String())
	}
	var items []CommunityBan
	if err := json.Unmarshal(bans.Body.Bytes(), &items); err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].User.ID != app.user.ID {
		t.Fatalf("ban list = %+v", items)
	}

	unbanned := app.request(t, http.MethodDelete, "/api/admin/bans/"+strconv.FormatInt(app.user.ID, 10), app.devToken, nil)
	if unbanned.Code != http.StatusOK {
		t.Fatalf("unban status = %d: %s", unbanned.Code, unbanned.Body.String())
	}
	recreated := app.request(t, http.MethodPost, "/api/blueprints", app.token, blueprintInput{
		Kind: "request", Title: "解除后重新提交", Body: "解除封禁后应可以继续发布",
	})
	if recreated.Code != http.StatusCreated {
		t.Fatalf("create after unban status = %d: %s", recreated.Code, recreated.Body.String())
	}
}

func TestDeveloperCanDeleteCommentAndBanAuthor(t *testing.T) {
	app := newTestApp(t)
	created := app.request(t, http.MethodPost, "/api/blueprints/1/comments", app.token, map[string]string{"body": "需要开发者处理的评论"})
	if created.Code != http.StatusCreated {
		t.Fatalf("create comment status = %d: %s", created.Code, created.Body.String())
	}
	var payload struct {
		Comment Comment `json:"comment"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	path := "/api/blueprints/1/comments/" + strconv.FormatInt(payload.Comment.ID, 10)
	deleted := app.request(t, http.MethodDelete, path, app.devToken, map[string]bool{"banAuthor": true})
	if deleted.Code != http.StatusOK {
		t.Fatalf("delete comment status = %d: %s", deleted.Code, deleted.Body.String())
	}
	blocked := app.request(t, http.MethodPost, "/api/blueprints", app.token, blueprintInput{
		Kind: "request", Title: "评论封禁后的请求", Body: "统一封禁时不应创建",
	})
	if blocked.Code != http.StatusForbidden {
		t.Fatalf("request after comment ban status = %d: %s", blocked.Code, blocked.Body.String())
	}
}

func TestLoginRefreshesDeveloperFromRoleID(t *testing.T) {
	app := newTestApp(t)
	profile := userInfo{Subject: "role-user", Name: "角色用户", Roles: []roleInfo{{ID: "developer", Name: "开发者"}}}

	user, err := app.auth.upsertUser(context.Background(), "https://auth.example.test", profile, idTokenClaims{})
	if err != nil {
		t.Fatal(err)
	}
	if !user.IsDeveloper {
		t.Fatal("matching role_id should grant developer access")
	}

	profile.Roles = nil
	user, err = app.auth.upsertUser(context.Background(), "https://auth.example.test", profile, idTokenClaims{})
	if err != nil {
		t.Fatal(err)
	}
	if user.IsDeveloper {
		t.Fatal("a later login without role_id should revoke developer access")
	}

	var stored int
	if err := app.store.db.QueryRow(`SELECT is_developer FROM users WHERE issuer = ? AND subject = ?`, "https://auth.example.test", "role-user").Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored != 0 {
		t.Fatalf("stored is_developer = %d, want 0", stored)
	}
}
