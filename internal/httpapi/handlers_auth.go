package httpapi

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"

	"mymoments/internal/auth"
	"mymoments/internal/model"
	"mymoments/internal/store"
)

type userEnvelope struct {
	User        model.User         `json:"user"`
	Household   *model.Household   `json:"household,omitempty"`
	Preferences *model.Preferences `json:"preferences,omitempty"`
}

func (s *Server) handleBootstrap(w http.ResponseWriter, r *http.Request) {
	household, err := s.store.GetHousehold(r.Context())
	if err != nil && err != store.ErrNotFound {
		writeError(w, serverError(err))
		return
	}
	resp := map[string]any{
		"needsSetup": household == nil,
		"household":  household,
		"user":       nil,
	}
	if u, ok := currentUser(r); ok {
		resp["user"] = u.Public()
	}
	writeJSON(w, http.StatusOK, resp)
}

type setupRequest struct {
	HouseholdName string `json:"householdName"`
	Username      string `json:"username"`
	DisplayName   string `json:"displayName"`
	Password      string `json:"password"`
}

func (s *Server) handleSetup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if _, err := s.store.GetHousehold(ctx); err == nil {
		writeError(w, conflict("家庭已完成初始化"))
		return
	} else if err != store.ErrNotFound {
		writeError(w, serverError(err))
		return
	}
	var req setupRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	req.Username = strings.ToLower(auth.NormalizeUsername(req.Username))
	req.DisplayName = strings.TrimSpace(req.DisplayName)
	req.HouseholdName = strings.TrimSpace(req.HouseholdName)
	if !auth.ValidateUsername(req.Username) {
		writeError(w, badRequest("用户名需为 3–32 位字母、数字、下划线或短横线"))
		return
	}
	if req.DisplayName == "" || len([]rune(req.DisplayName)) > 40 {
		writeError(w, badRequest("请填写 1–40 个字符的称呼"))
		return
	}
	if err := auth.ValidatePassword(req.Password); err != nil {
		writeError(w, &APIError{Status: http.StatusBadRequest, Code: "auth.weak_password", Message: err.Error()})
		return
	}
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	name := req.HouseholdName
	if name == "" {
		name = "我的家"
	}
	var userID int64
	created := false
	var user *model.User
	err = s.store.Tx(ctx, func(tx *sql.Tx) error {
		ok, err := store.CreateHouseholdIfAbsent(ctx, tx, name, s.cfg.DefaultTZ)
		if err != nil {
			return err
		}
		if !ok {
			return errSetupConflict
		}
		created = true
		id, err := store.CreateUserTx(ctx, tx, &model.User{
			Username:           req.Username,
			DisplayName:        req.DisplayName,
			PasswordHash:       hash,
			Role:               "admin",
			Status:             "active",
			MustChangePassword: false,
		})
		if err != nil {
			return err
		}
		userID = id
		return nil
	})
	if err != nil {
		if err == errSetupConflict {
			writeError(w, conflict("家庭已完成初始化"))
			return
		}
		writeError(w, serverError(err))
		return
	}
	if !created {
		writeError(w, conflict("家庭已完成初始化"))
		return
	}
	user, err = s.store.GetUserByID(ctx, userID)
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	if err := s.startSession(w, r, user.ID); err != nil {
		writeError(w, serverError(err))
		return
	}
	household, _ := s.store.GetHousehold(ctx)
	writeJSON(w, http.StatusCreated, userEnvelope{User: user.Public(), Household: household})
}

var errSetupConflict = &APIError{Status: http.StatusConflict, Code: "setup.conflict", Message: "家庭已完成初始化"}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	username := strings.ToLower(auth.NormalizeUsername(req.Username))
	if username == "" || req.Password == "" {
		writeError(w, badRequest("请输入用户名和密码"))
		return
	}
	key := clientIP(r, s.cfg.TrustedProxyHeader) + "|" + username
	if !s.limiter.allow(key) {
		w.Header().Set("Retry-After", "300")
		writeError(w, &APIError{Status: http.StatusTooManyRequests, Code: "auth.rate_limited", Message: "登录尝试过于频繁，请稍后再试"})
		return
	}
	user, err := s.store.GetUserByUsername(r.Context(), username)
	if err != nil || !auth.VerifyPassword(user.PasswordHash, req.Password) {
		s.limiter.fail(key)
		writeError(w, &APIError{Status: http.StatusUnauthorized, Code: "auth.invalid_credentials", Message: "用户名或密码不正确"})
		return
	}
	if user.Status != "active" {
		s.limiter.fail(key)
		writeError(w, forbidden("账号已停用，请联系家庭管理员"))
		return
	}
	s.limiter.success(key)
	if err := s.startSession(w, r, user.ID); err != nil {
		writeError(w, serverError(err))
		return
	}
	household, _ := s.store.GetHousehold(r.Context())
	writeJSON(w, http.StatusOK, userEnvelope{User: user.Public(), Household: household})
}

func (s *Server) handleSession(w http.ResponseWriter, r *http.Request) {
	user, ok := currentUser(r)
	if !ok {
		writeError(w, unauthenticated("尚未登录"))
		return
	}
	household, err := s.store.GetHousehold(r.Context())
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	prefs, err := s.store.GetPreferences(r.Context(), user.ID)
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	writeJSON(w, http.StatusOK, userEnvelope{User: user.Public(), Household: household, Preferences: prefs})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if id := currentSessionID(r); id != "" {
		_ = s.store.DeleteSession(r.Context(), id)
	}
	s.clearSessionCookie(w, r)
	w.WriteHeader(http.StatusNoContent)
}

type passwordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

func (s *Server) handlePasswordChange(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r)
	var req passwordRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	if !auth.VerifyPassword(user.PasswordHash, req.CurrentPassword) {
		writeError(w, &APIError{Status: http.StatusBadRequest, Code: "auth.invalid_credentials", Message: "当前密码不正确"})
		return
	}
	if req.NewPassword == req.CurrentPassword {
		writeError(w, badRequest("新密码不能与当前密码相同"))
		return
	}
	if err := auth.ValidatePassword(req.NewPassword); err != nil {
		writeError(w, &APIError{Status: http.StatusBadRequest, Code: "auth.weak_password", Message: err.Error()})
		return
	}
	hash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	if err := s.store.UpdatePassword(r.Context(), user.ID, hash, false); err != nil {
		writeError(w, serverError(err))
		return
	}
	_ = s.store.DeleteUserSessionsExcept(r.Context(), user.ID, currentSessionID(r))
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r)
	prefs, err := s.store.GetPreferences(r.Context(), user.ID)
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	writeJSON(w, http.StatusOK, userEnvelope{User: user.Public(), Preferences: prefs})
}

type mePatchRequest struct {
	DisplayName *string `json:"displayName"`
	Relation    *string `json:"relation"`
	AvatarKey   *string `json:"avatarKey"`
}

func (s *Server) handleMePatch(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r)
	var req mePatchRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	if req.DisplayName != nil {
		name := strings.TrimSpace(*req.DisplayName)
		if name == "" || len([]rune(name)) > 40 {
			writeError(w, badRequest("请填写 1–40 个字符的称呼"))
			return
		}
		user.DisplayName = name
	}
	if req.Relation != nil {
		rel := strings.TrimSpace(*req.Relation)
		if len([]rune(rel)) > 40 {
			writeError(w, badRequest("关系描述过长"))
			return
		}
		if rel == "" {
			user.Relation = nil
		} else {
			user.Relation = &rel
		}
	}
	if req.AvatarKey != nil {
		key := strings.TrimSpace(*req.AvatarKey)
		if key == "" {
			user.AvatarKey = nil
		} else {
			user.AvatarKey = &key
		}
	}
	if err := s.store.UpdateUser(r.Context(), user); err != nil {
		writeError(w, serverError(err))
		return
	}
	writeJSON(w, http.StatusOK, userEnvelope{User: user.Public()})
}

func (s *Server) handlePreferences(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r)
	prefs, err := s.store.GetPreferences(r.Context(), user.ID)
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	writeJSON(w, http.StatusOK, prefs)
}

type preferencesPatch struct {
	Locale            *string  `json:"locale"`
	DateFormat        *string  `json:"dateFormat"`
	Dashboard         *string  `json:"dashboard"`
	WeatherCity       *string  `json:"weatherCity"`
	WeatherLat        *float64 `json:"weatherLat"`
	WeatherLon        *float64 `json:"weatherLon"`
	WeatherAuto       *bool    `json:"weatherAuto"`
	NotificationPrefs *string  `json:"notificationPrefs"`
	Style             *string  `json:"style"`
}

func (s *Server) handlePreferencesPatch(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r)
	var req preferencesPatch
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	prefs, err := s.store.GetPreferences(r.Context(), user.ID)
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	if req.Locale != nil {
		prefs.Locale = *req.Locale
	}
	if req.DateFormat != nil {
		if *req.DateFormat != "24h" && *req.DateFormat != "12h" {
			writeError(w, badRequest("时间格式仅支持 24h 或 12h"))
			return
		}
		prefs.DateFormat = *req.DateFormat
	}
	if req.Dashboard != nil {
		if err := validateJSONField(*req.Dashboard); err != nil {
			writeError(w, badRequest("卡片布局格式不正确"))
			return
		}
		prefs.Dashboard = req.Dashboard
	}
	if req.WeatherCity != nil {
		city := strings.TrimSpace(*req.WeatherCity)
		if city == "" {
			prefs.WeatherCity = nil
		} else {
			prefs.WeatherCity = &city
		}
	}
	if req.WeatherLat != nil {
		prefs.WeatherLat = req.WeatherLat
	}
	if req.WeatherLon != nil {
		prefs.WeatherLon = req.WeatherLon
	}
	if req.WeatherAuto != nil {
		prefs.WeatherAuto = *req.WeatherAuto
	}
	if req.NotificationPrefs != nil {
		if err := validateJSONField(*req.NotificationPrefs); err != nil {
			writeError(w, badRequest("通知偏好格式不正确"))
			return
		}
		prefs.NotificationPrefs = req.NotificationPrefs
	}
	if req.Style != nil {
		style := strings.TrimSpace(*req.Style)
		if style == "" {
			prefs.Style = nil
		} else {
			if err := validateJSONField(style); err != nil {
				writeError(w, badRequest("样式格式不正确"))
				return
			}
			prefs.Style = &style
		}
	}
	if err := s.store.UpdatePreferences(r.Context(), prefs); err != nil {
		writeError(w, serverError(err))
		return
	}
	writeJSON(w, http.StatusOK, prefs)
}

func validateJSONField(v string) error {
	if v == "" {
		return nil
	}
	if !json.Valid([]byte(v)) {
		return errInvalidJSON
	}
	return nil
}

var errInvalidJSON = &APIError{Status: http.StatusBadRequest, Code: "validation.invalid_field", Message: "JSON 格式不正确"}
