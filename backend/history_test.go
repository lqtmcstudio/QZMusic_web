package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestGitHubHistorySyncCachesMasterCommits(t *testing.T) {
	app := newTestApp(t)
	var calls atomic.Int32
	github := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Query string `json:"query"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(request.Query, `ref(qualifiedName: "refs/heads/master")`) {
			t.Fatalf("history query is not pinned to master: %s", request.Query)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Fatalf("authorization header = %q", r.Header.Get("Authorization"))
		}
		call := calls.Add(1)
		sha := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		title := "first commit"
		committedAt := "2026-07-14T12:00:00Z"
		if call > 1 {
			sha = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
			title = "new commit"
			committedAt = "2026-07-14T13:00:00Z"
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"data": map[string]any{
				"repository": map[string]any{
					"ref": map[string]any{
						"target": map[string]any{
							"history": map[string]any{
								"nodes": []map[string]any{{
									"oid": sha, "messageHeadline": title, "committedDate": committedAt,
									"additions": 11225, "deletions": 1475, "changedFiles": 32,
									"url":    "https://github.com/example/repo/commit/" + sha,
									"author": map[string]any{"name": "QZ Dev", "user": map[string]any{"login": "qzdev", "avatarUrl": "https://example.com/avatar.png"}},
								}},
								"pageInfo": map[string]any{"hasNextPage": false, "endCursor": ""},
							},
						},
					},
				},
			},
		})
	}))
	defer github.Close()

	cfg := Config{
		GitHubAPIKey:      "test-token",
		GitHubGraphQLURL:  github.URL,
		GitHubAndroidRepo: "example/repo",
		GitHubWindowsRepo: "example/windows",
	}
	service := newHistoryService(cfg, app.store)
	repo := parseHistoryRepo("android", cfg.GitHubAndroidRepo)
	if err := service.syncRepo(context.Background(), repo); err != nil {
		t.Fatal(err)
	}
	if err := service.syncRepo(context.Background(), repo); err != nil {
		t.Fatal(err)
	}

	server := &Server{cfg: cfg, store: app.store, auth: app.auth}
	items, err := server.queryHistory(context.Background(), "android", 50, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("cached commits = %d, want 2", len(items))
	}
	if items[0].Title != "new commit" || items[0].FilesChanged != 32 || items[0].Additions != 11225 || items[0].Deletions != 1475 {
		t.Fatalf("newest commit = %+v", items[0])
	}
	state := server.queryHistoryState(context.Background(), "android")
	if state.LastSuccess == "" || state.Error != "" {
		t.Fatalf("sync state = %+v", state)
	}
}
