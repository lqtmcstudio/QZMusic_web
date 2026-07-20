package main

import (
	"bufio"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata"
)

type Config struct {
	HTTPAddr           string
	AppURL             string
	WebDist            string
	DatabasePath       string
	AuthIssuer         string
	OAuthClientID      string
	OAuthClientSecret  string
	OAuthRedirectURI   string
	OAuthScopes        string
	DeveloperRoleID    string
	SessionDuration    time.Duration
	CookieSecure       bool
	QuotaLocation      *time.Location
	GitHubAPIKey       string
	GitHubGraphQLURL   string
	GitHubAndroidRepo  string
	GitHubWindowsRepo  string
	GitHubSyncInterval time.Duration
	AllowedOrigin      string
}

func loadConfig() (Config, error) {
	_ = loadEnvFile(".env")

	sessionDays := envInt("SESSION_DAYS", 14)
	locationName := env("QUOTA_TIMEZONE", "Asia/Shanghai")
	location, err := time.LoadLocation(locationName)
	if err != nil {
		return Config{}, fmt.Errorf("load quota timezone: %w", err)
	}
	appURL := strings.TrimRight(env("APP_URL", "http://localhost:5173"), "/")
	parsedAppURL, err := url.Parse(appURL)
	if err != nil || parsedAppURL.Scheme == "" || parsedAppURL.Host == "" {
		return Config{}, errors.New("APP_URL must be an absolute URL")
	}

	cfg := Config{
		HTTPAddr:           env("HTTP_ADDR", ":8787"),
		AppURL:             appURL,
		WebDist:            env("WEB_DIST", "../dist"),
		DatabasePath:       env("DATABASE_PATH", "./data/qz-music.db"),
		AuthIssuer:         strings.TrimRight(env("AUTH_ISSUER", "https://auth.re-link.top"), "/"),
		OAuthClientID:      os.Getenv("OAUTH_CLIENT_ID"),
		OAuthClientSecret:  os.Getenv("OAUTH_CLIENT_SECRET"),
		OAuthRedirectURI:   env("OAUTH_REDIRECT_URI", "http://localhost:8787/auth/callback"),
		OAuthScopes:        env("OAUTH_SCOPES", "openid profile email"),
		DeveloperRoleID:    env("DEVELOPER_ROLE_ID", "developer"),
		SessionDuration:    time.Duration(sessionDays) * 24 * time.Hour,
		CookieSecure:       envBool("COOKIE_SECURE", strings.EqualFold(parsedAppURL.Scheme, "https")),
		QuotaLocation:      location,
		GitHubAPIKey:       os.Getenv("GITHUB_API_KEY"),
		GitHubGraphQLURL:   env("GITHUB_GRAPHQL_URL", "https://api.github.com/graphql"),
		GitHubAndroidRepo:  env("GITHUB_ANDROID_REPO", "nevodev/QZ-Music"),
		GitHubWindowsRepo:  env("GITHUB_WINDOWS_REPO", "lqtmcstudio/QZMusic_PC"),
		GitHubSyncInterval: time.Duration(envInt("GITHUB_SYNC_INTERVAL_MINUTES", 10)) * time.Minute,
		AllowedOrigin:      parsedAppURL.Scheme + "://" + parsedAppURL.Host,
	}
	return cfg, nil
}

func loadEnvFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), "\"'")
		if _, exists := os.LookupEnv(key); !exists {
			_ = os.Setenv(key, value)
		}
	}
	return scanner.Err()
}

func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) int {
	value, err := strconv.Atoi(os.Getenv(key))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func envBool(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}
