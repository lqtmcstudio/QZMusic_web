package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct {
	db       *sql.DB
	location *time.Location
}

func openStore(path string, location *time.Location) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create data directory: %w", err)
	}
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(path)+"?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(8)
	store := &Store{db: db, location: location}
	if err := store.migrate(context.Background()); err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) migrate(ctx context.Context) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			issuer TEXT NOT NULL,
			subject TEXT NOT NULL,
			name TEXT NOT NULL,
			username TEXT NOT NULL DEFAULT '',
			email TEXT NOT NULL DEFAULT '',
			picture TEXT NOT NULL DEFAULT '',
			is_developer INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			last_seen_at TEXT NOT NULL,
			UNIQUE(issuer, subject)
		)`,
		`CREATE TABLE IF NOT EXISTS oauth_states (
			state_hash TEXT PRIMARY KEY,
			nonce TEXT NOT NULL,
			verifier TEXT NOT NULL,
			return_to TEXT NOT NULL,
			expires_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS sessions (
			token_hash TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			expires_at TEXT NOT NULL,
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS blueprints (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			kind TEXT NOT NULL CHECK(kind IN ('preview','request')),
			status TEXT NOT NULL CHECK(status IN ('request','in_progress','voting','deprecated','released')),
			title TEXT NOT NULL,
			body TEXT NOT NULL,
			progress INTEGER NOT NULL DEFAULT 0 CHECK(progress BETWEEN 0 AND 100),
			author_id INTEGER NOT NULL REFERENCES users(id),
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS blueprint_images (
			blueprint_id INTEGER NOT NULL REFERENCES blueprints(id) ON DELETE CASCADE,
			position INTEGER NOT NULL,
			url TEXT NOT NULL,
			PRIMARY KEY(blueprint_id, position)
		)`,
		`CREATE TABLE IF NOT EXISTS updates (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			body TEXT NOT NULL,
			scope TEXT NOT NULL DEFAULT '',
			author_id INTEGER NOT NULL REFERENCES users(id),
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS comments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			entity_type TEXT NOT NULL CHECK(entity_type IN ('blueprints','updates')),
			entity_id INTEGER NOT NULL,
			author_id INTEGER NOT NULL REFERENCES users(id),
			body TEXT NOT NULL,
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS likes (
			entity_type TEXT NOT NULL CHECK(entity_type IN ('blueprints','updates')),
			entity_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			created_at TEXT NOT NULL,
			PRIMARY KEY(entity_type, entity_id, user_id)
		)`,
		`CREATE TABLE IF NOT EXISTS votes (
			blueprint_id INTEGER NOT NULL REFERENCES blueprints(id) ON DELETE CASCADE,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			choice TEXT NOT NULL CHECK(choice IN ('want','dont_want')),
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			PRIMARY KEY(blueprint_id, user_id)
		)`,
		`CREATE TABLE IF NOT EXISTS daily_usage (
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			day_key TEXT NOT NULL,
			comments INTEGER NOT NULL DEFAULT 0,
			requests INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY(user_id, day_key)
		)`,
		`CREATE TABLE IF NOT EXISTS blueprint_bans (
			user_id INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
			banned_by INTEGER NOT NULL REFERENCES users(id),
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS community_bans (
			user_id INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
			banned_by INTEGER NOT NULL REFERENCES users(id),
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS github_commits (
			platform TEXT NOT NULL CHECK(platform IN ('android','windows')),
			repo_full_name TEXT NOT NULL,
			sha TEXT NOT NULL,
			title TEXT NOT NULL,
			body TEXT NOT NULL DEFAULT '',
			author_name TEXT NOT NULL DEFAULT '',
			author_login TEXT NOT NULL DEFAULT '',
			author_avatar TEXT NOT NULL DEFAULT '',
			committed_at TEXT NOT NULL,
			files_changed INTEGER NOT NULL DEFAULT 0,
			additions INTEGER NOT NULL DEFAULT 0,
			deletions INTEGER NOT NULL DEFAULT 0,
			html_url TEXT NOT NULL,
			cached_at TEXT NOT NULL,
			PRIMARY KEY(repo_full_name, sha)
		)`,
		`CREATE TABLE IF NOT EXISTS github_sync_state (
			platform TEXT PRIMARY KEY CHECK(platform IN ('android','windows')),
			repo_full_name TEXT NOT NULL,
			last_checked TEXT NOT NULL DEFAULT '',
			last_success TEXT NOT NULL DEFAULT '',
			last_error TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE INDEX IF NOT EXISTS idx_blueprints_status_updated ON blueprints(status, updated_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_comments_entity ON comments(entity_type, entity_id, created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_comments_author_created ON comments(author_id, created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_expiry ON sessions(expires_at)`,
				`CREATE TABLE IF NOT EXISTS site_config (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			updated_by INTEGER REFERENCES users(id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_github_commits_platform_time ON github_commits(platform, committed_at DESC)`,
	}
	for _, statement := range statements {
		if _, err := s.db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}
	// Preserve bans created by the earlier blueprint-only moderation schema.
	if _, err := s.db.ExecContext(ctx, `INSERT OR IGNORE INTO community_bans (user_id, banned_by, created_at) SELECT user_id, banned_by, created_at FROM blueprint_bans`); err != nil {
		return fmt.Errorf("migrate legacy bans: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, `DROP TABLE blueprint_bans`); err != nil {
		return fmt.Errorf("remove legacy bans: %w", err)
	}
	// Add scope column to updates if missing (for existing databases).
	s.db.ExecContext(ctx, `ALTER TABLE updates ADD COLUMN scope TEXT NOT NULL DEFAULT ''`)
	return nil
}

func (s *Store) dailyLimits(ctx context.Context, userID int64) (DailyLimits, error) {
	if userID == 0 {
		return DailyLimits{CommentsRemaining: 10, RequestsRemaining: 2}, nil
	}
	dayKey := time.Now().In(s.location).Format("2006-01-02")
	var comments, requests int
	err := s.db.QueryRowContext(ctx, `SELECT comments, requests FROM daily_usage WHERE user_id = ? AND day_key = ?`, userID, dayKey).Scan(&comments, &requests)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return DailyLimits{}, err
	}
	return DailyLimits{CommentsRemaining: max(0, 10-comments), RequestsRemaining: max(0, 2-requests)}, nil
}

var errDailyLimit = errors.New("daily limit reached")

func (s *Store) consumeDailyQuota(ctx context.Context, tx *sql.Tx, userID int64, quota string) error {
	dayKey := time.Now().In(s.location).Format("2006-01-02")
	if _, err := tx.ExecContext(ctx, `INSERT INTO daily_usage (user_id, day_key, comments, requests) VALUES (?, ?, 0, 0) ON CONFLICT(user_id, day_key) DO NOTHING`, userID, dayKey); err != nil {
		return err
	}
	query := ""
	switch quota {
	case "comments":
		query = `UPDATE daily_usage SET comments = comments + 1 WHERE user_id = ? AND day_key = ? AND comments < 10`
	case "requests":
		query = `UPDATE daily_usage SET requests = requests + 1 WHERE user_id = ? AND day_key = ? AND requests < 2`
	default:
		return errors.New("unknown daily quota")
	}
	result, err := tx.ExecContext(ctx, query, userID, dayKey)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return errDailyLimit
	}
	return nil
}

func (s *Store) isCommunityBanned(ctx context.Context, userID int64) (bool, error) {
	var banned bool
	err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM community_bans WHERE user_id = ?)`, userID).Scan(&banned)
	return banned, err
}

func displayTime(value string, location *time.Location) string {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return value
	}
	local := parsed.In(location)
	now := time.Now().In(location)
	if local.Year() == now.Year() && local.YearDay() == now.YearDay() {
		return "今天 " + local.Format("15:04")
	}
	if local.Year() == now.Year() {
		return local.Format("01月02日")
	}
	return local.Format("2006年01月02日")
}

func validEntityType(value string) bool { return value == "blueprints" || value == "updates" }

func normalizeScope(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "frontend", "vue", "前端":
		return "frontend"
	case "backend", "go", "后端":
		return "backend"
	case "android", "安卓":
		return "android"
	case "pc", "windows", "pc端":
		return "pc"
	case "all", "全端":
		return "all"
	default:
		return ""
	}
}

func (s *Store) entityExists(ctx context.Context, entityType string, id int64) (bool, error) {
	if !validEntityType(entityType) {
		return false, nil
	}
	query := `SELECT EXISTS(SELECT 1 FROM blueprints WHERE id = ?)`
	if entityType == "updates" {
		query = `SELECT EXISTS(SELECT 1 FROM updates WHERE id = ?)`
	}
	var exists bool
	return exists, s.db.QueryRowContext(ctx, query, id).Scan(&exists)
}

func trimRequired(value string, maxLength int) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", errors.New("内容不能为空")
	}
	if len([]rune(value)) > maxLength {
		return "", fmt.Errorf("内容不能超过 %d 个字符", maxLength)
	}
	return value, nil
}
