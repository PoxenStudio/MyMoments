package httpapi

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"time"

	"mymoments/internal/auth"
	"mymoments/internal/model"
)

// tokenScopeCatalog 定义 Token 可授权的模块与操作。
var tokenScopeCatalog = map[string][]string{
	"tasks":     {"read", "write", "delete"},
	"calendar":  {"read", "write", "delete"},
	"birthdays": {"read", "write", "delete"},
	"contacts":  {"read", "write", "delete"},
	"messages":  {"read", "write", "delete"},
	"notes":     {"read", "write", "delete"},
	"family":    {"read"},
	"timeline":  {"read"},
	"reminders": {"read"},
	"weather":   {"read"},
}

type tokenView struct {
	ID         int64               `json:"id"`
	Name       string              `json:"name"`
	Prefix     string              `json:"prefix"`
	Scopes     map[string][]string `json:"scopes"`
	ExpiresAt  *int64              `json:"expiresAt"`
	LastUsedAt *int64              `json:"lastUsedAt"`
	CreatedAt  int64               `json:"createdAt"`
}

func viewToken(t model.APIToken) tokenView {
	scopes := map[string][]string{}
	_ = json.Unmarshal([]byte(t.Scopes), &scopes)
	return tokenView{
		ID: t.ID, Name: t.Name, Prefix: t.Prefix, Scopes: scopes,
		ExpiresAt: t.ExpiresAt, LastUsedAt: t.LastUsedAt, CreatedAt: t.CreatedAt,
	}
}

func (s *Server) handleTokensList(w http.ResponseWriter, r *http.Request) {
	u, _ := currentUser(r)
	items, err := s.store.ListTokensByUser(r.Context(), u.ID)
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	out := make([]tokenView, 0, len(items))
	for _, item := range items {
		out = append(out, viewToken(item))
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

type tokenCreateRequest struct {
	Name          string              `json:"name"`
	Scopes        map[string][]string `json:"scopes"`
	ExpiresInDays *int                `json:"expiresInDays"`
}

func (s *Server) handleTokenCreate(w http.ResponseWriter, r *http.Request) {
	u, _ := currentUser(r)
	var req tokenCreateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" || len([]rune(name)) > 60 {
		writeError(w, badRequest("Token 名称需为 1–60 个字符"))
		return
	}
	scopes, msg := normalizeTokenScopes(req.Scopes)
	if msg != "" {
		writeError(w, badRequest(msg))
		return
	}

	days := 90
	if req.ExpiresInDays != nil {
		days = *req.ExpiresInDays
	}
	var expiresAt *int64
	if days != 0 {
		switch days {
		case 30, 90, 180, 365:
		default:
			writeError(w, badRequest("有效期只能为 30、90、180、365 天或永不过期"))
			return
		}
		ts := time.Now().AddDate(0, 0, days).Unix()
		expiresAt = &ts
	}

	raw, err := auth.RandomToken(32)
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	raw = "mm_" + raw
	encoded, err := json.Marshal(scopes)
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	prefix := raw
	if len(prefix) > 10 {
		prefix = prefix[:10]
	}
	token := model.APIToken{
		UserID: u.ID, Name: name, Hash: auth.HashToken(raw), Prefix: prefix,
		Scopes: string(encoded), ExpiresAt: expiresAt, CreatedAt: time.Now().Unix(),
	}
	if err := s.store.CreateToken(r.Context(), &token); err != nil {
		writeError(w, serverError(err))
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"token": raw, "item": viewToken(token)})
}

func (s *Server) handleTokenDelete(w http.ResponseWriter, r *http.Request) {
	u, _ := currentUser(r)
	id, err := parseID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	if err := s.store.RevokeToken(r.Context(), u.ID, id); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func normalizeTokenScopes(in map[string][]string) (map[string][]string, string) {
	out := map[string][]string{}
	for module, actions := range in {
		allowed, ok := tokenScopeCatalog[module]
		if !ok {
			return nil, "未知的权限模块：" + module
		}
		seen := map[string]bool{}
		clean := make([]string, 0, len(actions))
		for _, action := range actions {
			if !containsString(allowed, action) {
				return nil, "模块 " + module + " 不支持操作：" + action
			}
			if seen[action] {
				continue
			}
			seen[action] = true
			clean = append(clean, action)
		}
		if len(clean) == 0 {
			continue
		}
		sort.Strings(clean)
		out[module] = clean
	}
	if len(out) == 0 {
		return nil, "至少需要授予一个权限"
	}
	return out, ""
}

func containsString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
