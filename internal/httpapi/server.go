package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"log/slog"
	"mime"
	"net/http"
	"path"
	"strings"
	"time"

	"mymoments/internal/config"
	"mymoments/internal/store"
	"mymoments/internal/webui"
)

// Server 持有 API 依赖并注册路由。
type Server struct {
	cfg     config.Config
	store   *store.Store
	web     fs.FS
	limiter *loginLimiter
}

// New 构造 HTTP 服务。
func New(cfg config.Config, st *store.Store) (*Server, error) {
	web, err := webui.FS()
	if err != nil {
		return nil, err
	}
	return &Server{cfg: cfg, store: st, web: web, limiter: newLoginLimiter(8, 5*time.Minute)}, nil
}

// Handler 返回完整中间件链。
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	s.routes(mux)
	return s.middleware(s.resolveUser(mux))
}

func (s *Server) routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /readyz", s.handleReady)
	mux.HandleFunc("GET /openapi.yaml", s.handleOpenAPI)

	api := "/api/v1"
	mux.HandleFunc("GET "+api+"/bootstrap", s.handleBootstrap)
	mux.HandleFunc("POST "+api+"/setup", s.handleSetup)
	mux.HandleFunc("POST "+api+"/auth/login", s.handleLogin)
	mux.HandleFunc("GET "+api+"/auth/session", s.handleSession)
	mux.HandleFunc("POST "+api+"/auth/logout", s.handleLogout)

	mux.Handle("POST "+api+"/auth/password", s.requireAuth(http.HandlerFunc(s.handlePasswordChange)))
	mux.Handle("GET "+api+"/me", s.requireAuth(http.HandlerFunc(s.handleMe)))
	mux.Handle("PATCH "+api+"/me", s.requireAuth(http.HandlerFunc(s.handleMePatch)))
	mux.Handle("GET "+api+"/me/preferences", s.requireAuth(http.HandlerFunc(s.handlePreferences)))
	mux.Handle("PATCH "+api+"/me/preferences", s.requireAuth(http.HandlerFunc(s.handlePreferencesPatch)))

	mux.Handle("GET "+api+"/family/members", s.requireAuth(http.HandlerFunc(s.handleMembers)))
	mux.Handle("POST "+api+"/family/members", s.requireAdmin(http.HandlerFunc(s.handleMemberCreate)))
	mux.Handle("PATCH "+api+"/family/members/{id}", s.requireAuth(http.HandlerFunc(s.handleMemberPatch)))
	mux.Handle("POST "+api+"/family/members/{id}/reset-password", s.requireAdmin(http.HandlerFunc(s.handleMemberResetPassword)))

	mux.Handle("GET "+api+"/household", s.requireAuth(http.HandlerFunc(s.handleHousehold)))
	mux.Handle("PATCH "+api+"/household", s.requireAdmin(http.HandlerFunc(s.handleHouseholdPatch)))

	mux.Handle("GET "+api+"/tasks", s.requireAuth(http.HandlerFunc(s.handleTasksList)))
	mux.Handle("POST "+api+"/tasks", s.requireAuth(http.HandlerFunc(s.handleTaskCreate)))
	mux.Handle("GET "+api+"/tasks/{id}", s.requireAuth(http.HandlerFunc(s.handleTaskGet)))
	mux.Handle("PATCH "+api+"/tasks/{id}", s.requireAuth(http.HandlerFunc(s.handleTaskPatch)))
	mux.Handle("DELETE "+api+"/tasks/{id}", s.requireAuth(http.HandlerFunc(s.handleTaskDelete)))
	mux.Handle("POST "+api+"/tasks/{id}/completion", s.requireAuth(http.HandlerFunc(s.handleTaskComplete)))
	mux.Handle("DELETE "+api+"/tasks/{id}/completion", s.requireAuth(http.HandlerFunc(s.handleTaskUndo)))
	mux.Handle("PATCH "+api+"/tasks/{id}/subtasks/{subId}", s.requireAuth(http.HandlerFunc(s.handleSubtaskPatch)))
	mux.Handle("GET "+api+"/tasks/{id}/media", s.requireAuth(http.HandlerFunc(s.handleTaskMediaList)))
	mux.Handle("POST "+api+"/tasks/{id}/media", s.requireAuth(http.HandlerFunc(s.handleMediaUpload)))

	mux.Handle("GET "+api+"/templates", s.requireAuth(http.HandlerFunc(s.handleTemplatesList)))
	mux.Handle("POST "+api+"/templates", s.requireAuth(http.HandlerFunc(s.handleTemplateCreate)))
	mux.Handle("PATCH "+api+"/templates/{id}", s.requireAuth(http.HandlerFunc(s.handleTemplatePatch)))
	mux.Handle("DELETE "+api+"/templates/{id}", s.requireAuth(http.HandlerFunc(s.handleTemplateDelete)))

	mux.Handle("GET "+api+"/reminders", s.requireAuth(http.HandlerFunc(s.handleRemindersList)))
	mux.Handle("PATCH "+api+"/reminders/{id}", s.requireAuth(http.HandlerFunc(s.handleReminderPatch)))

	mux.Handle("GET "+api+"/media/{id}", s.requireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.handleMediaGet(w, r, false)
	})))
	mux.Handle("GET "+api+"/media/{id}/thumbnail", s.requireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.handleMediaGet(w, r, true)
	})))
	mux.Handle("DELETE "+api+"/media/{id}", s.requireAuth(http.HandlerFunc(s.handleMediaDelete)))

	mux.HandleFunc("/", s.handleStatic)
}

func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	if err := s.store.IsReady(ctx); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"status": "not_ready", "reason": "database"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ready"})
}

func (s *Server) handleOpenAPI(w http.ResponseWriter, r *http.Request) {
	b, err := openAPISpec()
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	_, _ = w.Write(b)
}

// openAPISpec 在 M7 填充，当前返回最小桩。
func openAPISpec() ([]byte, error) {
	return []byte("openapi: 3.1.0\ninfo:\n  title: MyMoments API\n  version: v1\npaths: {}\n"), nil
}

func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	clean := path.Clean("/" + strings.TrimPrefix(r.URL.Path, "/"))
	if strings.HasPrefix(clean, "/api/") || clean == "/healthz" || clean == "/readyz" {
		http.NotFound(w, r)
		return
	}
	name := strings.TrimPrefix(clean, "/")
	if name == "" || !strings.Contains(path.Base(name), ".") {
		name = "index.html"
	}
	b, err := fs.ReadFile(s.web, name)
	if err != nil {
		name = "index.html"
		b, err = fs.ReadFile(s.web, name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
	}
	if ext := path.Ext(name); ext != "" {
		if ct := mime.TypeByExtension(ext); ct != "" {
			w.Header().Set("Content-Type", ct)
		}
	}
	if strings.HasPrefix(clean, "/assets/") {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	} else if name == "index.html" {
		w.Header().Set("Cache-Control", "no-cache")
	}
	if name == "sw.js" || name == "registerSW.js" {
		w.Header().Set("Cache-Control", "no-cache")
	}
	sum := sha256.Sum256(b)
	w.Header().Set("ETag", `"`+hex.EncodeToString(sum[:8])+`"`)
	if r.Header.Get("If-None-Match") == w.Header().Get("ETag") {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	_, _ = w.Write(b)
}

var _ = slog.Info
