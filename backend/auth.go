package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const userContextKey contextKey = "user"

type oidcDiscovery struct {
	Issuer                string `json:"issuer"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	UserInfoEndpoint      string `json:"userinfo_endpoint"`
	JWKSURI               string `json:"jwks_uri"`
}

type oauthPending struct {
	Nonce    string
	Verifier string
	ReturnTo string
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	IDToken     string `json:"id_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Scope       string `json:"scope"`
	Error       string `json:"error"`
	Description string `json:"error_description"`
}

type idTokenClaims struct {
	Name    string `json:"name"`
	Picture string `json:"picture"`
	Email   string `json:"email"`
	Nonce   string `json:"nonce"`
	jwt.RegisteredClaims
}

type userInfo struct {
	Subject  string     `json:"sub"`
	Name     string     `json:"name"`
	Username string     `json:"username"`
	Email    string     `json:"email"`
	Picture  string     `json:"picture"`
	Roles    []roleInfo `json:"roles"`
}

type roleInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type jwksDocument struct {
	Keys []jwk `json:"keys"`
}

type jwk struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Alg string `json:"alg"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
}

type authService struct {
	cfg         Config
	store       *Store
	httpClient  *http.Client
	mu          sync.Mutex
	discovery   oidcDiscovery
	discoveryAt time.Time
	jwks        jwksDocument
	jwksAt      time.Time
}

func newAuthService(cfg Config, store *Store) *authService {
	return &authService{cfg: cfg, store: store, httpClient: &http.Client{Timeout: 12 * time.Second}}
}

func (a *authService) configured() bool {
	return a.cfg.OAuthClientID != "" && a.cfg.OAuthClientSecret != "" && a.cfg.OAuthRedirectURI != ""
}

func (a *authService) discoveryDocument(ctx context.Context) (oidcDiscovery, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.discovery.Issuer != "" && time.Since(a.discoveryAt) < 30*time.Minute {
		return a.discovery, nil
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, a.cfg.AuthIssuer+"/.well-known/openid-configuration", nil)
	response, err := a.httpClient.Do(req)
	if err != nil {
		return oidcDiscovery{}, fmt.Errorf("load OIDC discovery: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return oidcDiscovery{}, fmt.Errorf("OIDC discovery returned %d", response.StatusCode)
	}
	var document oidcDiscovery
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&document); err != nil {
		return oidcDiscovery{}, err
	}
	if document.Issuer != a.cfg.AuthIssuer || document.AuthorizationEndpoint == "" || document.TokenEndpoint == "" || document.UserInfoEndpoint == "" || document.JWKSURI == "" {
		return oidcDiscovery{}, errors.New("OIDC discovery document is incomplete or issuer does not match")
	}
	a.discovery = document
	a.discoveryAt = time.Now()
	return document, nil
}

func (a *authService) jwksDocument(ctx context.Context, uri string, force bool) (jwksDocument, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if !force && len(a.jwks.Keys) > 0 && time.Since(a.jwksAt) < 15*time.Minute {
		return a.jwks, nil
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
	response, err := a.httpClient.Do(req)
	if err != nil {
		return jwksDocument{}, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return jwksDocument{}, fmt.Errorf("JWKS returned %d", response.StatusCode)
	}
	var document jwksDocument
	if err := json.NewDecoder(io.LimitReader(response.Body, 2<<20)).Decode(&document); err != nil {
		return jwksDocument{}, err
	}
	a.jwks = document
	a.jwksAt = time.Now()
	return document, nil
}

func randomURLSafe(bytes int) (string, error) {
	raw := make([]byte, bytes)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func hashToken(value string) string {
	hash := sha256.Sum256([]byte(value))
	return hex.EncodeToString(hash[:])
}

func safeReturnTo(value string) string {
	if value == "" || !strings.HasPrefix(value, "/") || strings.HasPrefix(value, "//") {
		return "/"
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.IsAbs() || parsed.Host != "" {
		return "/"
	}
	return value
}

func (a *authService) handleLogin(w http.ResponseWriter, r *http.Request) {
	if !a.configured() {
		writeError(w, http.StatusServiceUnavailable, "sso_not_configured", "登录系统尚未配置 OAuth Client")
		return
	}
	discovery, err := a.discoveryDocument(r.Context())
	if err != nil {
		writeError(w, http.StatusBadGateway, "sso_unavailable", "暂时无法连接统一认证服务")
		return
	}
	state, err := randomURLSafe(32)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "random_failed", "无法发起登录")
		return
	}
	nonce, _ := randomURLSafe(32)
	verifier, _ := randomURLSafe(64)
	challengeHash := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(challengeHash[:])
	expiresAt := time.Now().UTC().Add(8 * time.Minute).Format(time.RFC3339)
	_, err = a.store.db.ExecContext(r.Context(), `INSERT INTO oauth_states (state_hash, nonce, verifier, return_to, expires_at) VALUES (?, ?, ?, ?, ?)`, hashToken(state), nonce, verifier, safeReturnTo(r.URL.Query().Get("return_to")), expiresAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "state_failed", "无法发起登录")
		return
	}
	query := url.Values{
		"response_type":         {"code"},
		"client_id":             {a.cfg.OAuthClientID},
		"redirect_uri":          {a.cfg.OAuthRedirectURI},
		"scope":                 {a.cfg.OAuthScopes},
		"state":                 {state},
		"nonce":                 {nonce},
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
	}
	http.Redirect(w, r, discovery.AuthorizationEndpoint+"?"+query.Encode(), http.StatusFound)
}

func (a *authService) consumeOAuthState(ctx context.Context, state string) (oauthPending, error) {
	tx, err := a.store.db.BeginTx(ctx, nil)
	if err != nil {
		return oauthPending{}, err
	}
	defer tx.Rollback()
	var pending oauthPending
	var expiresAt string
	hash := hashToken(state)
	if err := tx.QueryRowContext(ctx, `SELECT nonce, verifier, return_to, expires_at FROM oauth_states WHERE state_hash = ?`, hash).Scan(&pending.Nonce, &pending.Verifier, &pending.ReturnTo, &expiresAt); err != nil {
		return oauthPending{}, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM oauth_states WHERE state_hash = ?`, hash); err != nil {
		return oauthPending{}, err
	}
	if err := tx.Commit(); err != nil {
		return oauthPending{}, err
	}
	expiry, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil || time.Now().After(expiry) {
		return oauthPending{}, errors.New("OAuth state expired")
	}
	return pending, nil
}

func (a *authService) handleCallback(w http.ResponseWriter, r *http.Request) {
	if oauthError := r.URL.Query().Get("error"); oauthError != "" {
		http.Redirect(w, r, a.cfg.AppURL+"/?auth_error="+url.QueryEscape(oauthError), http.StatusFound)
		return
	}
	code, state := r.URL.Query().Get("code"), r.URL.Query().Get("state")
	if code == "" || state == "" {
		writeError(w, http.StatusBadRequest, "invalid_callback", "登录回调缺少必要参数")
		return
	}
	pending, err := a.consumeOAuthState(r.Context(), state)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_state", "登录请求已过期，请重新登录")
		return
	}
	discovery, err := a.discoveryDocument(r.Context())
	if err != nil {
		writeError(w, http.StatusBadGateway, "sso_unavailable", "暂时无法连接统一认证服务")
		return
	}
	tokens, err := a.exchangeCode(r.Context(), discovery, code, pending.Verifier)
	if err != nil {
		writeError(w, http.StatusBadGateway, "token_exchange_failed", "登录凭证交换失败，请重新登录")
		return
	}
	claims, err := a.validateIDToken(r.Context(), discovery, tokens.IDToken, pending.Nonce)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid_id_token", "统一认证返回的身份凭证无效")
		return
	}
	profile, err := a.fetchUserInfo(r.Context(), discovery, tokens.AccessToken)
	if err != nil || profile.Subject == "" || profile.Subject != claims.Subject {
		writeError(w, http.StatusBadGateway, "userinfo_failed", "无法读取登录用户资料")
		return
	}
	user, err := a.upsertUser(r.Context(), discovery.Issuer, profile, claims)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "user_failed", "无法创建本地登录会话")
		return
	}
	if err := a.issueSession(w, r.Context(), user.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "session_failed", "无法创建本地登录会话")
		return
	}
	http.Redirect(w, r, a.cfg.AppURL+safeReturnTo(pending.ReturnTo), http.StatusFound)
}

func (a *authService) exchangeCode(ctx context.Context, discovery oidcDiscovery, code, verifier string) (tokenResponse, error) {
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {a.cfg.OAuthRedirectURI},
		"code_verifier": {verifier},
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, discovery.TokenEndpoint, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(a.cfg.OAuthClientID, a.cfg.OAuthClientSecret)
	response, err := a.httpClient.Do(req)
	if err != nil {
		return tokenResponse{}, err
	}
	defer response.Body.Close()
	var tokens tokenResponse
	if err := json.NewDecoder(io.LimitReader(response.Body, 2<<20)).Decode(&tokens); err != nil {
		return tokenResponse{}, err
	}
	if response.StatusCode != http.StatusOK || tokens.Error != "" || tokens.AccessToken == "" || tokens.IDToken == "" {
		return tokenResponse{}, errors.New("OAuth token exchange rejected")
	}
	return tokens, nil
}

func (a *authService) validateIDToken(ctx context.Context, discovery oidcDiscovery, rawToken, expectedNonce string) (idTokenClaims, error) {
	claims := idTokenClaims{}
	keyFunc := func(token *jwt.Token) (any, error) {
		kid, _ := token.Header["kid"].(string)
		if token.Method.Alg() != "RS256" || kid == "" {
			return nil, errors.New("unexpected signing algorithm")
		}
		for attempt := 0; attempt < 2; attempt++ {
			document, err := a.jwksDocument(ctx, discovery.JWKSURI, attempt == 1)
			if err != nil {
				return nil, err
			}
			for _, key := range document.Keys {
				if key.Kid == kid && key.Kty == "RSA" && (key.Alg == "" || key.Alg == "RS256") {
					return rsaPublicKey(key)
				}
			}
		}
		return nil, errors.New("signing key not found")
	}
	parsed, err := jwt.ParseWithClaims(rawToken, &claims, keyFunc,
		jwt.WithValidMethods([]string{"RS256"}),
		jwt.WithIssuer(discovery.Issuer),
		jwt.WithAudience(a.cfg.OAuthClientID),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithLeeway(60*time.Second),
	)
	if err != nil || !parsed.Valid || claims.Subject == "" {
		return idTokenClaims{}, errors.New("invalid ID token")
	}
	if subtle.ConstantTimeCompare([]byte(claims.Nonce), []byte(expectedNonce)) != 1 {
		return idTokenClaims{}, errors.New("nonce mismatch")
	}
	return claims, nil
}

func rsaPublicKey(key jwk) (*rsa.PublicKey, error) {
	modulusBytes, err := base64.RawURLEncoding.DecodeString(key.N)
	if err != nil {
		return nil, err
	}
	exponentBytes, err := base64.RawURLEncoding.DecodeString(key.E)
	if err != nil {
		return nil, err
	}
	exponent := 0
	for _, value := range exponentBytes {
		exponent = exponent<<8 + int(value)
	}
	if exponent == 0 {
		return nil, errors.New("invalid RSA exponent")
	}
	return &rsa.PublicKey{N: new(big.Int).SetBytes(modulusBytes), E: exponent}, nil
}

func (a *authService) fetchUserInfo(ctx context.Context, discovery oidcDiscovery, accessToken string) (userInfo, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, discovery.UserInfoEndpoint, nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	response, err := a.httpClient.Do(req)
	if err != nil {
		return userInfo{}, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return userInfo{}, errors.New("userinfo rejected")
	}
	var profile userInfo
	return profile, json.NewDecoder(io.LimitReader(response.Body, 2<<20)).Decode(&profile)
}

func (a *authService) upsertUser(ctx context.Context, issuer string, profile userInfo, claims idTokenClaims) (User, error) {
	name := strings.TrimSpace(profile.Name)
	if name == "" {
		name = strings.TrimSpace(claims.Name)
	}
	if name == "" {
		name = profile.Username
	}
	if name == "" {
		name = "QZ 用户"
	}
	picture := profile.Picture
	if picture == "" {
		picture = claims.Picture
	}
	email := profile.Email
	if email == "" {
		email = claims.Email
	}
	isDeveloper := false
	for _, role := range profile.Roles {
		if subtle.ConstantTimeCompare([]byte(strings.TrimSpace(role.ID)), []byte(a.cfg.DeveloperRoleID)) == 1 {
			isDeveloper = true
			break
		}
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := a.store.db.ExecContext(ctx, `INSERT INTO users (issuer, subject, name, username, email, picture, is_developer, created_at, last_seen_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(issuer, subject) DO UPDATE SET name = excluded.name, username = excluded.username, email = excluded.email, picture = excluded.picture, is_developer = excluded.is_developer, last_seen_at = excluded.last_seen_at`, issuer, profile.Subject, name, profile.Username, email, picture, boolInt(isDeveloper), now, now)
	if err != nil {
		return User{}, err
	}
	var user User
	var developer int
	err = a.store.db.QueryRowContext(ctx, `SELECT id, issuer, subject, name, username, email, picture, is_developer FROM users WHERE issuer = ? AND subject = ?`, issuer, profile.Subject).Scan(&user.ID, &user.Issuer, &user.Subject, &user.Name, &user.Username, &user.Email, &user.Picture, &developer)
	user.IsDeveloper = developer == 1
	return user, err
}

func (a *authService) issueSession(w http.ResponseWriter, ctx context.Context, userID int64) error {
	token, err := randomURLSafe(32)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	expires := now.Add(a.cfg.SessionDuration)
	_, err = a.store.db.ExecContext(ctx, `INSERT INTO sessions (token_hash, user_id, expires_at, created_at) VALUES (?, ?, ?, ?)`, hashToken(token), userID, expires.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{Name: "qz_session", Value: token, Path: "/", Expires: expires, MaxAge: int(a.cfg.SessionDuration.Seconds()), HttpOnly: true, Secure: a.cfg.CookieSecure, SameSite: http.SameSiteLaxMode})
	return nil
}

func (a *authService) currentUser(r *http.Request) *User {
	cookie, err := r.Cookie("qz_session")
	if err != nil || cookie.Value == "" {
		return nil
	}
	var user User
	var developer int
	var expiresAt string
	err = a.store.db.QueryRowContext(r.Context(), `SELECT u.id, u.issuer, u.subject, u.name, u.username, u.email, u.picture, u.is_developer, s.expires_at
		FROM sessions s JOIN users u ON u.id = s.user_id WHERE s.token_hash = ?`, hashToken(cookie.Value)).Scan(&user.ID, &user.Issuer, &user.Subject, &user.Name, &user.Username, &user.Email, &user.Picture, &developer, &expiresAt)
	if err != nil {
		return nil
	}
	expires, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil || time.Now().After(expires) {
		_, _ = a.store.db.ExecContext(r.Context(), `DELETE FROM sessions WHERE token_hash = ?`, hashToken(cookie.Value))
		return nil
	}
	user.IsDeveloper = developer == 1
	return &user
}

func (a *authService) withUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if user := a.currentUser(r); user != nil {
			r = r.WithContext(context.WithValue(r.Context(), userContextKey, user))
		}
		next.ServeHTTP(w, r)
	})
}

func userFromContext(ctx context.Context) *User {
	user, _ := ctx.Value(userContextKey).(*User)
	return user
}

func (a *authService) handleLogout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie("qz_session"); err == nil && cookie.Value != "" {
		_, _ = a.store.db.ExecContext(r.Context(), `DELETE FROM sessions WHERE token_hash = ?`, hashToken(cookie.Value))
	}
	http.SetCookie(w, &http.Cookie{Name: "qz_session", Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: a.cfg.CookieSecure, SameSite: http.SameSiteLaxMode})
	w.WriteHeader(http.StatusNoContent)
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func parseInt64(value string) (int64, error) {
	return strconv.ParseInt(value, 10, 64)
}
