package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const githubHistoryQuery = `query CommitHistory($owner: String!, $name: String!, $first: Int!, $after: String) {
  repository(owner: $owner, name: $name) {
    ref(qualifiedName: "refs/heads/master") {
      target {
        ... on Commit {
          history(first: $first, after: $after) {
            nodes {
              oid
              messageHeadline
              message
              committedDate
              additions
              deletions
              changedFiles
              url
              author {
                name
                user { login avatarUrl }
              }
            }
            pageInfo { hasNextPage endCursor }
          }
        }
      }
    }
  }
}`

type historyRepo struct {
	Platform string
	FullName string
	Owner    string
	Name     string
}

type HistoryService struct {
	cfg    Config
	store  *Store
	client *http.Client
	repos  []historyRepo
}

type GitHubCommit struct {
	Platform     string `json:"platform"`
	Repository   string `json:"repository"`
	SHA          string `json:"sha"`
	Title        string `json:"title"`
	Body         string `json:"body,omitempty"`
	AuthorName   string `json:"authorName"`
	AuthorLogin  string `json:"authorLogin,omitempty"`
	AuthorAvatar string `json:"authorAvatar,omitempty"`
	CommittedAt  string `json:"committedAt"`
	FilesChanged int    `json:"filesChanged"`
	Additions    int    `json:"additions"`
	Deletions    int    `json:"deletions"`
	URL          string `json:"url"`
}

type HistorySyncState struct {
	Platform    string `json:"platform"`
	Repository  string `json:"repository"`
	LastChecked string `json:"lastChecked,omitempty"`
	LastSuccess string `json:"lastSuccess,omitempty"`
	Error       string `json:"error,omitempty"`
	Configured  bool   `json:"configured"`
}

type githubGraphQLResponse struct {
	Data struct {
		Repository *struct {
			Ref *struct {
				Target struct {
					History struct {
						Nodes    []githubCommitNode `json:"nodes"`
						PageInfo struct {
							HasNextPage bool   `json:"hasNextPage"`
							EndCursor   string `json:"endCursor"`
						} `json:"pageInfo"`
					} `json:"history"`
				} `json:"target"`
			} `json:"ref"`
		} `json:"repository"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

type githubCommitNode struct {
	OID             string `json:"oid"`
	MessageHeadline string `json:"messageHeadline"`
	Message         string `json:"message"`
	CommittedDate   string `json:"committedDate"`
	Additions       int    `json:"additions"`
	Deletions       int    `json:"deletions"`
	ChangedFiles    int    `json:"changedFiles"`
	URL             string `json:"url"`
	Author          struct {
		Name string `json:"name"`
		User *struct {
			Login     string `json:"login"`
			AvatarURL string `json:"avatarUrl"`
		} `json:"user"`
	} `json:"author"`
}

func newHistoryService(cfg Config, store *Store) *HistoryService {
	return &HistoryService{
		cfg:    cfg,
		store:  store,
		client: &http.Client{Timeout: 25 * time.Second},
		repos: []historyRepo{
			parseHistoryRepo("android", cfg.GitHubAndroidRepo),
			parseHistoryRepo("windows", cfg.GitHubWindowsRepo),
		},
	}
}

func parseHistoryRepo(platform, fullName string) historyRepo {
	owner, name, ok := strings.Cut(strings.TrimSpace(fullName), "/")
	if !ok || strings.Contains(name, "/") {
		return historyRepo{Platform: platform, FullName: strings.TrimSpace(fullName)}
	}
	return historyRepo{Platform: platform, FullName: owner + "/" + name, Owner: owner, Name: name}
}

func (h *HistoryService) Start(ctx context.Context) {
	go func() {
		h.refreshAll(ctx)
		ticker := time.NewTicker(h.cfg.GitHubSyncInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				h.refreshAll(ctx)
			}
		}
	}()
}

func (h *HistoryService) refreshAll(ctx context.Context) {
	for _, repo := range h.repos {
		if err := h.syncRepo(ctx, repo); err != nil && !errors.Is(err, context.Canceled) {
			log.Printf("GitHub history sync failed for %s: %v", repo.Platform, err)
		}
	}
}

func (h *HistoryService) syncRepo(ctx context.Context, repo historyRepo) error {
	if repo.Owner == "" || repo.Name == "" {
		err := errors.New("仓库地址格式无效")
		h.recordSyncFailure(ctx, repo, err)
		return err
	}
	if strings.TrimSpace(h.cfg.GitHubAPIKey) == "" {
		err := errors.New("尚未配置 GITHUB_API_KEY")
		h.recordSyncFailure(ctx, repo, err)
		return err
	}

	var cachedCount int
	_ = h.store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM github_commits WHERE repo_full_name = ?`, repo.FullName).Scan(&cachedCount)
	maxPages := 1
	if cachedCount == 0 {
		maxPages = 10
	}

	after := ""
	for page := 0; page < maxPages; page++ {
		result, err := h.fetchCommitPage(ctx, repo, after)
		if err != nil {
			h.recordSyncFailure(ctx, repo, err)
			return err
		}
		if err := h.cacheCommitPage(ctx, repo, result.Data.Repository.Ref.Target.History.Nodes); err != nil {
			h.recordSyncFailure(ctx, repo, err)
			return err
		}
		pageInfo := result.Data.Repository.Ref.Target.History.PageInfo
		if !pageInfo.HasNextPage || pageInfo.EndCursor == "" {
			break
		}
		after = pageInfo.EndCursor
	}
	h.recordSyncSuccess(ctx, repo)
	return nil
}

func (h *HistoryService) fetchCommitPage(ctx context.Context, repo historyRepo, after string) (githubGraphQLResponse, error) {
	variables := map[string]any{"owner": repo.Owner, "name": repo.Name, "first": 100, "after": nil}
	if after != "" {
		variables["after"] = after
	}
	payload, _ := json.Marshal(map[string]any{"query": githubHistoryQuery, "variables": variables})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.cfg.GitHubGraphQLURL, bytes.NewReader(payload))
	if err != nil {
		return githubGraphQLResponse{}, err
	}
	req.Header.Set("Authorization", "Bearer "+h.cfg.GitHubAPIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "QZ-Music-Website")
	response, err := h.client.Do(req)
	if err != nil {
		return githubGraphQLResponse{}, fmt.Errorf("连接 GitHub 失败: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return githubGraphQLResponse{}, fmt.Errorf("GitHub 返回 HTTP %d", response.StatusCode)
	}
	var result githubGraphQLResponse
	if err := json.NewDecoder(io.LimitReader(response.Body, 4<<20)).Decode(&result); err != nil {
		return githubGraphQLResponse{}, errors.New("GitHub 返回了无法解析的数据")
	}
	if len(result.Errors) > 0 {
		message := strings.TrimSpace(result.Errors[0].Message)
		if len([]rune(message)) > 240 {
			message = string([]rune(message)[:240])
		}
		return githubGraphQLResponse{}, errors.New(message)
	}
	if result.Data.Repository == nil {
		return githubGraphQLResponse{}, errors.New("GitHub 仓库不存在或 Token 无权访问")
	}
	if result.Data.Repository.Ref == nil {
		return githubGraphQLResponse{}, errors.New("GitHub 仓库没有 master 分支")
	}
	return result, nil
}

func (h *HistoryService) cacheCommitPage(ctx context.Context, repo historyRepo, nodes []githubCommitNode) error {
	tx, err := h.store.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	cachedAt := time.Now().UTC().Format(time.RFC3339)
	for _, node := range nodes {
		if strings.TrimSpace(node.OID) == "" || strings.TrimSpace(node.CommittedDate) == "" {
			continue
		}
		authorLogin, authorAvatar := "", ""
		if node.Author.User != nil {
			authorLogin = node.Author.User.Login
			authorAvatar = node.Author.User.AvatarURL
		}
		body := extractCommitBody(node.Message, node.MessageHeadline)
		_, err := tx.ExecContext(ctx, `INSERT INTO github_commits (
			platform, repo_full_name, sha, title, body, author_name, author_login, author_avatar,
			committed_at, files_changed, additions, deletions, html_url, cached_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(repo_full_name, sha) DO UPDATE SET
			title = excluded.title,
			body = excluded.body,
			author_name = excluded.author_name,
			author_login = excluded.author_login,
			author_avatar = excluded.author_avatar,
			committed_at = excluded.committed_at,
			files_changed = excluded.files_changed,
			additions = excluded.additions,
			deletions = excluded.deletions,
			html_url = excluded.html_url,
			cached_at = excluded.cached_at`,
			repo.Platform, repo.FullName, node.OID, strings.TrimSpace(node.MessageHeadline), body, strings.TrimSpace(node.Author.Name), authorLogin, authorAvatar,
			node.CommittedDate, max(0, node.ChangedFiles), max(0, node.Additions), max(0, node.Deletions), node.URL, cachedAt)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func extractCommitBody(message, headline string) string {
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		return ""
	}
	headline = strings.TrimSpace(headline)
	if headline == "" {
		return trimmed
	}
	// The full message starts with the headline, then optionally a blank line and the body.
	rest := trimmed
	if strings.HasPrefix(rest, headline) {
		rest = rest[len(headline):]
	}
	rest = strings.TrimLeft(rest, "\r\n")
	rest = strings.TrimSpace(rest)
	return rest
}

func (h *HistoryService) recordSyncFailure(ctx context.Context, repo historyRepo, syncErr error) {
	message := syncErr.Error()
	if len([]rune(message)) > 300 {
		message = string([]rune(message)[:300])
	}
	_, _ = h.store.db.ExecContext(ctx, `INSERT INTO github_sync_state (platform, repo_full_name, last_checked, last_success, last_error)
		VALUES (?, ?, ?, '', ?)
		ON CONFLICT(platform) DO UPDATE SET repo_full_name = excluded.repo_full_name, last_checked = excluded.last_checked, last_error = excluded.last_error`,
		repo.Platform, repo.FullName, time.Now().UTC().Format(time.RFC3339), message)
}

func (h *HistoryService) recordSyncSuccess(ctx context.Context, repo historyRepo) {
	now := time.Now().UTC().Format(time.RFC3339)
	_, _ = h.store.db.ExecContext(ctx, `INSERT INTO github_sync_state (platform, repo_full_name, last_checked, last_success, last_error)
		VALUES (?, ?, ?, ?, '')
		ON CONFLICT(platform) DO UPDATE SET repo_full_name = excluded.repo_full_name, last_checked = excluded.last_checked, last_success = excluded.last_success, last_error = ''`,
		repo.Platform, repo.FullName, now, now)
}

func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	platform := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("platform")))
	if platform == "" {
		platform = "android"
	}
	if platform != "android" && platform != "windows" {
		writeError(w, http.StatusBadRequest, "invalid_platform", "平台只能是 android 或 windows")
		return
	}
	limit := historyQueryInt(r.URL.Query().Get("limit"), 50, 1, 200)
	offset := historyQueryInt(r.URL.Query().Get("offset"), 0, 0, 10000)
	items, err := s.queryHistory(r.Context(), platform, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "history_failed", "无法读取项目更新历史")
		return
	}
	state := s.queryHistoryState(r.Context(), platform)
	state.Configured = strings.TrimSpace(s.cfg.GitHubAPIKey) != ""
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "sync": state})
}

func historyQueryInt(raw string, fallback, minimum, maximum int) int {
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return min(maximum, max(minimum, value))
}

func (s *Server) queryHistory(ctx context.Context, platform string, limit, offset int) ([]GitHubCommit, error) {
	rows, err := s.store.db.QueryContext(ctx, `SELECT platform, repo_full_name, sha, title, body, author_name, author_login, author_avatar,
		committed_at, files_changed, additions, deletions, html_url
		FROM github_commits WHERE platform = ? ORDER BY committed_at DESC LIMIT ? OFFSET ?`, platform, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]GitHubCommit, 0)
	for rows.Next() {
		var item GitHubCommit
		if err := rows.Scan(&item.Platform, &item.Repository, &item.SHA, &item.Title, &item.Body, &item.AuthorName, &item.AuthorLogin, &item.AuthorAvatar,
			&item.CommittedAt, &item.FilesChanged, &item.Additions, &item.Deletions, &item.URL); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Server) queryHistoryState(ctx context.Context, platform string) HistorySyncState {
	repository := s.cfg.GitHubAndroidRepo
	if platform == "windows" {
		repository = s.cfg.GitHubWindowsRepo
	}
	state := HistorySyncState{Platform: platform, Repository: repository}
	_ = s.store.db.QueryRowContext(ctx, `SELECT repo_full_name, last_checked, last_success, last_error FROM github_sync_state WHERE platform = ?`, platform).
		Scan(&state.Repository, &state.LastChecked, &state.LastSuccess, &state.Error)
	return state
}
