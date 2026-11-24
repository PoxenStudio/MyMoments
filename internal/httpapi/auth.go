package httpapi

import (
	"context"
	"net/http"
	"strings"
	"time"

	"mymoments/internal/auth"
	"mymoments/internal/model"
)

const sessionCookie = "mymoments_session"

func currentUser(r *http.Request) (*model.User, bool) {
	u, ok := r.Context().Value(ctxUser).(*model.User)
	return u, ok
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
		next.ServeHTTP(w, r)
	})
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
			if u, tokenID, ok := s.resolveBearer(ctx, r); ok {
				ctx = context.WithValue(ctx, ctxUser, u)
				ctx = context.WithValue(ctx, ctxTokenID, tokenID)
				ctx = context.WithValue(ctx, ctxViaToken, true)
			}
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// resolveBearer 在 M7 接入 API Token 时实现；当前返回匿名。
func (s *Server) resolveBearer(ctx context.Context, r *http.Request) (*model.User, int64, bool) {
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		return nil, 0, false
	}
	token, err := s.store.GetTokenByHash(ctx, auth.HashToken(strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))))
	if err != nil {
		return nil, 0, false
	}
	u, err := s.store.GetUserByID(ctx, token.UserID)
	if err != nil || u.Status != "active" {
		return nil, 0, false
	}
	_ = s.store.TouchToken(ctx, token.ID)
	return u, token.ID, true
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
	raw, err := auth.RandomToken(32)
	if err != nil {
		return err
	}
	now := time.Now()
	sess := model.Session{
		ID:         auth.HashToken(raw),
		UserID:     userID,
		ExpiresAt:  now.Add(s.cfg.SessionTTL).Unix(),
		CreatedAt:  now.Unix(),
		LastSeenAt: now.Unix(),
		UserAgent:  truncate(r.UserAgent(), 255),
		IP:         truncate(clientIP(r, s.cfg.TrustedProxyHeader), 64),
	}
	if err := s.store.CreateSession(r.Context(), &sess); err != nil {
		return err
	}
	s.setSessionCookie(w, r, raw, s.cfg.SessionTTL)
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
