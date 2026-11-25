package httpapi

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"io/fs"
	"log/slog"
	"mime"
	"net/http"
	"path"
	"strings"
	"time"

	"mymoments/internal/config"
	"mymoments/internal/holiday"
	"mymoments/internal/store"
	"mymoments/internal/weather"
	"mymoments/internal/webui"
)

// Server 持有 API 依赖并注册路由。
type Server struct {
	cfg      config.Config
	store    *store.Store
	web      fs.FS
	limiter  *loginLimiter
	weather  *weather.Manager
	holidays *holiday.Service
}

// New 构造 HTTP 服务。
func New(cfg config.Config, st *store.Store) (*Server, error) {
	web, err := webui.FS()
	if err != nil {
		return nil, err
	}
	openMeteo := weather.New(cfg.WeatherBaseURL, cfg.WeatherAPIKey, 10*time.Minute)
	qweather := weather.NewQWeather(cfg.QWeatherBaseURL, cfg.QWeatherKey, 10*time.Minute)
	return &Server{
		cfg:      cfg,
		store:    st,
		web:      web,
		limiter:  newLoginLimiter(8, 5*time.Minute),
		weather:  weather.NewManager(openMeteo, qweather, cfg.WeatherProvider),
		holidays: holiday.NewService(cfg.HolidayDir(), cfg.HolidayDataURL),
	}, nil
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
	mux.Handle("POST "+api+"/me/avatar", s.requireAuth(http.HandlerFunc(s.handleAvatarUpload)))
	mux.Handle("GET "+api+"/avatars/{key}", s.requireAuth(http.HandlerFunc(s.handleAvatarGet)))

	mux.Handle("GET "+api+"/family/members", s.requireAuth(http.HandlerFunc(s.handleMembers)))
	mux.Handle("POST "+api+"/family/members", s.requireAdmin(http.HandlerFunc(s.handleMemberCreate)))
	mux.Handle("PATCH "+api+"/family/members/{id}", s.requireAuth(http.HandlerFunc(s.handleMemberPatch)))
	mux.Handle("GET "+api+"/family/members/{id}/style", s.requireAdmin(http.HandlerFunc(s.handleMemberStyleGet)))
	mux.Handle("PATCH "+api+"/family/members/{id}/style", s.requireAdmin(http.HandlerFunc(s.handleMemberStylePatch)))
	mux.Handle("POST "+api+"/family/members/{id}/avatar", s.requireAdmin(http.HandlerFunc(s.handleMemberAvatarUpload)))
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

	mux.Handle("GET "+api+"/events", s.requireAuth(http.HandlerFunc(s.handleEventsList)))
	mux.Handle("POST "+api+"/events", s.requireAuth(http.HandlerFunc(s.handleEventCreate)))
	mux.Handle("GET "+api+"/events/{id}", s.requireAuth(http.HandlerFunc(s.handleEventGet)))
	mux.Handle("PATCH "+api+"/events/{id}", s.requireAuth(http.HandlerFunc(s.handleEventPatch)))
	mux.Handle("DELETE "+api+"/events/{id}", s.requireAuth(http.HandlerFunc(s.handleEventDelete)))

	mux.Handle("GET "+api+"/contacts", s.requireAuth(http.HandlerFunc(s.handleContactsList)))
	mux.Handle("POST "+api+"/contacts", s.requireAuth(http.HandlerFunc(s.handleContactCreate)))
	mux.Handle("POST "+api+"/contacts/import", s.requireAuth(http.HandlerFunc(s.handleContactsImport)))
	mux.Handle("POST "+api+"/contacts/avatar", s.requireAuth(http.HandlerFunc(s.handleContactAvatarUpload)))
	mux.Handle("GET "+api+"/contacts/{id}", s.requireAuth(http.HandlerFunc(s.handleContactGet)))
	mux.Handle("PATCH "+api+"/contacts/{id}", s.requireAuth(http.HandlerFunc(s.handleContactPatch)))
	mux.Handle("DELETE "+api+"/contacts/{id}", s.requireAuth(http.HandlerFunc(s.handleContactDelete)))
	mux.Handle("GET "+api+"/contacts/{id}/associations", s.requireAuth(http.HandlerFunc(s.handleContactAssociations)))
	mux.Handle("PUT "+api+"/contacts/{id}/pinned", s.requireAuth(http.HandlerFunc(s.handleContactPin)))
	mux.Handle("POST "+api+"/contacts/cleanup", s.requireAuth(http.HandlerFunc(s.handleContactsCleanup)))

	mux.Handle("GET "+api+"/birthdays", s.requireAuth(http.HandlerFunc(s.handleBirthdaysList)))
	mux.Handle("POST "+api+"/birthdays", s.requireAuth(http.HandlerFunc(s.handleBirthdayCreate)))
	mux.Handle("PATCH "+api+"/birthdays/{id}", s.requireAuth(http.HandlerFunc(s.handleBirthdayPatch)))
	mux.Handle("DELETE "+api+"/birthdays/{id}", s.requireAuth(http.HandlerFunc(s.handleBirthdayDelete)))

	mux.Handle("GET "+api+"/notes", s.requireAuth(http.HandlerFunc(s.handleNotesList)))
	mux.Handle("POST "+api+"/notes", s.requireAuth(http.HandlerFunc(s.handleNoteCreate)))
	mux.Handle("GET "+api+"/notes/{id}", s.requireAuth(http.HandlerFunc(s.handleNoteGet)))
	mux.Handle("PATCH "+api+"/notes/{id}", s.requireAuth(http.HandlerFunc(s.handleNotePatch)))
	mux.Handle("DELETE "+api+"/notes/{id}", s.requireAuth(http.HandlerFunc(s.handleNoteDelete)))

	mux.Handle("GET "+api+"/messages", s.requireAuth(http.HandlerFunc(s.handleMessagesList)))
	mux.Handle("POST "+api+"/messages", s.requireAuth(http.HandlerFunc(s.handleMessageCreate)))
	mux.Handle("GET "+api+"/messages/{id}", s.requireAuth(http.HandlerFunc(s.handleMessageGet)))
	mux.Handle("PATCH "+api+"/messages/{id}", s.requireAuth(http.HandlerFunc(s.handleMessagePatch)))
	mux.Handle("DELETE "+api+"/messages/{id}", s.requireAuth(http.HandlerFunc(s.handleMessageDelete)))
	mux.Handle("POST "+api+"/messages/{id}/replies", s.requireAuth(http.HandlerFunc(s.handleMessageReply)))

	mux.Handle("GET "+api+"/timeline", s.requireAuth(http.HandlerFunc(s.handleTimeline)))

	mux.Handle("GET "+api+"/dashboard", s.requireAuth(http.HandlerFunc(s.handleDashboard)))
	mux.Handle("GET "+api+"/dashboard/layout", s.requireAuth(http.HandlerFunc(s.handleDashboardLayout)))
	mux.Handle("PUT "+api+"/dashboard/layout", s.requireAuth(http.HandlerFunc(s.handleDashboardLayoutPut)))
	mux.Handle("PUT "+api+"/household/default-layout", s.requireAdmin(http.HandlerFunc(s.handleHouseholdDefaultLayout)))
	mux.Handle("GET "+api+"/weather", s.requireAuth(http.HandlerFunc(s.handleWeather)))
	mux.Handle("GET "+api+"/weather/sources", s.requireAuth(http.HandlerFunc(s.handleWeatherSources)))
	mux.Handle("GET "+api+"/calendar/meta", s.requireAuth(http.HandlerFunc(s.handleCalendarMeta)))
	mux.Handle("GET "+api+"/search", s.requireAuth(http.HandlerFunc(s.handleSearch)))

	mux.Handle("GET "+api+"/tokens", s.requireAuth(http.HandlerFunc(s.handleTokensList)))
	mux.Handle("POST "+api+"/tokens", s.requireAuth(http.HandlerFunc(s.handleTokenCreate)))
	mux.Handle("DELETE "+api+"/tokens/{id}", s.requireAuth(http.HandlerFunc(s.handleTokenDelete)))

	mux.Handle("GET "+api+"/notifications", s.requireAuth(http.HandlerFunc(s.handleNotificationsList)))
	mux.Handle("POST "+api+"/notifications/read-all", s.requireAuth(http.HandlerFunc(s.handleNotificationsReadAll)))
	mux.Handle("PATCH "+api+"/notifications/{id}/read", s.requireAuth(http.HandlerFunc(s.handleNotificationRead)))

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

//go:embed openapi.yaml
var openAPIBytes []byte

// openAPISpec 返回内置的 OpenAPI 文档。
func openAPISpec() ([]byte, error) {
	return openAPIBytes, nil
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
