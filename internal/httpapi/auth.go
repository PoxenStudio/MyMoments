package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"mymoments/internal/auth"
	"mymoments/internal/model"
)

const sessionCookie = "mymoments_session"

// rememberSessionTTL 是「记住登录」选项对应的免登时长。
const rememberSessionTTL = 7 * 24 * time.Hour

func currentUser(r *http.Request) (*model.User, bool) {
	u, ok := r.Context().Value(ctxUser).(*model.User)
	return u, ok
}

func currentToken(r *http.Request) (*model.APIToken, bool) {
	t, ok := r.Context().Value(ctxToken).(*model.APIToken)
	return t, ok
}

func currentSessionID(r *http.Request) string {
	v, _ := r.Context().Value(ctxSessionID).(string)
	return v
}

func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, ok := currentUser(r)
		if !ok {
			writeError(w, unauthenticated("请先登录"))
			return
		}
		if u.MustChangePassword && !passwordChangeAllowed(r.URL.Path) {
			writeError(w, &APIError{Status: http.StatusForbidden, Code: "auth.password_change_required", Message: "请先设置新密码"})
			return
		}
		token, viaToken := currentToken(r)
		if !viaToken {
			next.ServeHTTP(w, r)
			return
		}
		module, action, allowed := tokenAccessRule(r)
		if !allowed || !tokenAllows(token, module, action) {
			_ = s.store.RecordTokenAudit(r.Context(), token.ID, module+":"+action, "denied", truncate(r.UserAgent(), 120))
			writeError(w, &APIError{Status: http.StatusForbidden, Code: "permission.denied", Message: "Token 无权执行此操作"})
			return
		}
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		result := "ok"
		if rec.status == http.StatusForbidden || rec.status == http.StatusUnauthorized {
			result = "denied"
		} else if rec.status >= 500 {
			result = "error"
		}
		_ = s.store.RecordTokenAudit(r.Context(), token.ID, module+":"+action, result, truncate(r.UserAgent(), 120))
	})
}

var tokenReadOnlyModules = map[string]bool{
	"family": true, "timeline": true, "reminders": true, "weather": true,
}

// searchTokenModules 是跨模块搜索涉及的内容模块。
var searchTokenModules = []string{"tasks", "calendar", "contacts", "messages", "notes"}

// tokenAccessRule 把请求映射到 Token scope 的模块与操作。
func tokenAccessRule(r *http.Request) (string, string, bool) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/")
	segment := path
	if i := strings.Index(segment, "/"); i >= 0 {
		segment = segment[:i]
	}
	var module string
	switch segment {
	case "tasks", "templates", "media":
		module = "tasks"
	case "events":
		module = "calendar"
	case "birthdays":
		module = "birthdays"
	case "contacts":
		module = "contacts"
	case "family", "household":
		module = "family"
	case "messages":
		module = "messages"
	case "notes":
		module = "notes"
	case "timeline":
		module = "timeline"
	case "reminders":
		module = "reminders"
	case "weather":
		module = "weather"
	case "search":
		module = "search"
	default:
		return "", "", false
	}
	var action string
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		action = "read"
	case http.MethodDelete:
		action = "delete"
	case http.MethodPost, http.MethodPut, http.MethodPatch:
		action = "write"
	default:
		return "", "", false
	}
	return module, action, true
}

func tokenAllows(token *model.APIToken, module, action string) bool {
	if token == nil || module == "" {
		return false
	}
	if module == "search" {
		return action == "read" && tokenAllowsSearch(token)
	}
	if tokenReadOnlyModules[module] && action != "read" {
		return false
	}
	var scopes map[string][]string
	if err := json.Unmarshal([]byte(token.Scopes), &scopes); err != nil {
		return false
	}
	for _, allowed := range scopes[module] {
		if allowed == action {
			return true
		}
	}
	return false
}

// tokenAllowsSearch 在 Token 至少拥有一个可搜索模块的读权限时放行搜索。
func tokenAllowsSearch(token *model.APIToken) bool {
	for _, module := range searchTokenModules {
		if tokenAllows(token, module, "read") {
			return true
		}
	}
	return false
}

func passwordChangeAllowed(path string) bool {
	switch path {
	case "/api/v1/auth/password", "/api/v1/auth/logout", "/api/v1/auth/session", "/api/v1/bootstrap", "/api/v1/me", "/api/v1/me/preferences":
		return true
	default:
		return false
	}
}

func (s *Server) requireAdmin(next http.Handler) http.Handler {
	return s.requireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, _ := currentUser(r)
		if u.Role != "admin" {
			writeError(w, forbidden("仅管理员可以执行此操作"))
			return
		}
		next.ServeHTTP(w, r)
	}))
}

// resolveUser 解析 Cookie 会话或 Bearer Token，匿名请求继续放行。
func (s *Server) resolveUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if cookie, err := r.Cookie(sessionCookie); err == nil && cookie.Value != "" {
			if sess, err := s.store.GetSession(ctx, auth.HashToken(cookie.Value)); err == nil {
				if u, err := s.store.GetUserByID(ctx, sess.UserID); err == nil && u.Status == "active" {
					ctx = context.WithValue(ctx, ctxUser, u)
					ctx = context.WithValue(ctx, ctxSessionID, sess.ID)
					if time.Now().Unix()-sess.LastSeenAt > 300 {
						_ = s.store.TouchSession(ctx, sess.ID, time.Now().Unix())
					}
				}
			}
		}
		if _, ok := currentUser(r.WithContext(ctx)); !ok {
			if u, token, ok := s.resolveBearer(ctx, r); ok {
				ctx = context.WithValue(ctx, ctxUser, u)
				ctx = context.WithValue(ctx, ctxTokenID, token.ID)
				ctx = context.WithValue(ctx, ctxToken, token)
				ctx = context.WithValue(ctx, ctxViaToken, true)
			}
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) resolveBearer(ctx context.Context, r *http.Request) (*model.User, *model.APIToken, bool) {
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		return nil, nil, false
	}
	token, err := s.store.GetTokenByHash(ctx, auth.HashToken(strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))))
	if err != nil {
		return nil, nil, false
	}
	u, err := s.store.GetUserByID(ctx, token.UserID)
	if err != nil || u.Status != "active" {
		return nil, nil, false
	}
	_ = s.store.TouchToken(ctx, token.ID)
	return u, token, true
}

func (s *Server) setSessionCookie(w http.ResponseWriter, r *http.Request, raw string, ttl time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    raw,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   requestIsSecure(r),
		MaxAge:   int(ttl.Seconds()),
	})
}

func (s *Server) clearSessionCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   requestIsSecure(r),
		MaxAge:   -1,
	})
}

func requestIsSecure(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	return strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

func (s *Server) startSession(w http.ResponseWriter, r *http.Request, userID int64) error {
	return s.startSessionWith(w, r, userID, s.cfg.SessionTTL, true)
}

// startSessionWith 创建服务端会话；persistent 为 false 时只下发浏览器会话级 Cookie。
func (s *Server) startSessionWith(w http.ResponseWriter, r *http.Request, userID int64, ttl time.Duration, persistent bool) error {
	raw, err := auth.RandomToken(32)
	if err != nil {
		return err
	}
	now := time.Now()
	sess := model.Session{
		ID:         auth.HashToken(raw),
		UserID:     userID,
		ExpiresAt:  now.Add(ttl).Unix(),
		CreatedAt:  now.Unix(),
		LastSeenAt: now.Unix(),
		UserAgent:  truncate(r.UserAgent(), 255),
		IP:         truncate(clientIP(r, s.cfg.TrustedProxyHeader), 64),
	}
	if err := s.store.CreateSession(r.Context(), &sess); err != nil {
		return err
	}
	if persistent {
		s.setSessionCookie(w, r, raw, ttl)
	} else {
		s.setSessionCookie(w, r, raw, 0)
	}
	return nil
}

func truncate(v string, max int) string {
	if len(v) <= max {
		return v
	}
	return v[:max]
}

func parseID(r *http.Request, name string) (int64, error) {
	raw := r.PathValue(name)
	var id int64
	for _, ch := range raw {
		if ch < '0' || ch > '9' {
			return 0, badRequest("无效的 ID")
		}
		id = id*10 + int64(ch-'0')
	}
	if id <= 0 {
		return 0, badRequest("无效的 ID")
	}
	return id, nil
}
