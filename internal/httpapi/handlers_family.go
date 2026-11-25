package httpapi

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"mymoments/internal/auth"
	"mymoments/internal/model"
	"mymoments/internal/store"
)

func (s *Server) handleMembers(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListUsers(r.Context())
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

type createMemberRequest struct {
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
	Role        string `json:"role"`
	Relation    string `json:"relation"`
	Password    string `json:"password"`
}

func (s *Server) handleMemberCreate(w http.ResponseWriter, r *http.Request) {
	var req createMemberRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	username := strings.ToLower(auth.NormalizeUsername(req.Username))
	displayName := strings.TrimSpace(req.DisplayName)
	if !auth.ValidateUsername(username) {
		writeError(w, badRequest("用户名需为 3–32 位字母、数字、下划线或短横线"))
		return
	}
	if displayName == "" || len([]rune(displayName)) > 40 {
		writeError(w, badRequest("请填写 1–40 个字符的称呼"))
		return
	}
	role := req.Role
	if role == "" {
		role = "member"
	}
	if role != "admin" && role != "member" {
		writeError(w, badRequest("角色只能是管理员或成员"))
		return
	}
	password := req.Password
	temporary := ""
	if password == "" {
		var err error
		temporary, err = auth.GenerateTemporaryPassword()
		if err != nil {
			writeError(w, serverError(err))
			return
		}
		password = temporary
	} else if err := auth.ValidatePassword(password); err != nil {
		writeError(w, &APIError{Status: http.StatusBadRequest, Code: "auth.weak_password", Message: err.Error()})
		return
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	count, err := s.store.CountUsers(r.Context())
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	u := &model.User{
		Username:           username,
		DisplayName:        displayName,
		PasswordHash:       hash,
		Role:               role,
		Status:             "active",
		MustChangePassword: true,
		SortOrder:          count,
	}
	if rel := strings.TrimSpace(req.Relation); rel != "" {
		u.Relation = &rel
	}
	var id int64
	err = s.store.Tx(r.Context(), func(tx *sql.Tx) error {
		newID, err := store.CreateUserTx(r.Context(), tx, u)
		if err != nil {
			return err
		}
		id = newID
		return nil
	})
	if err != nil {
		if strings.Contains(err.Error(), "username exists") {
			writeError(w, &APIError{Status: http.StatusConflict, Code: "username.taken", Message: "该用户名已被使用"})
			return
		}
		writeError(w, serverError(err))
		return
	}
	created, err := s.store.GetUserByID(r.Context(), id)
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"user":              created.Public(),
		"temporaryPassword": temporary,
	})
}

type memberPatchRequest struct {
	DisplayName *string `json:"displayName"`
	Relation    *string `json:"relation"`
	AvatarKey   *string `json:"avatarKey"`
	Role        *string `json:"role"`
	Status      *string `json:"status"`
}

func (s *Server) handleMemberPatch(w http.ResponseWriter, r *http.Request) {
	actor, _ := currentUser(r)
	id, err := parseID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	target, err := s.store.GetUserByID(r.Context(), id)
	if err != nil {
		writeError(w, notFound("成员不存在"))
		return
	}
	var req memberPatchRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	isSelf := actor.ID == target.ID
	if actor.Role != "admin" && !isSelf {
		writeError(w, forbidden("仅管理员可以修改其他成员"))
		return
	}
	if actor.Role != "admin" && (req.Role != nil || req.Status != nil) {
		writeError(w, forbidden("只有管理员可以调整角色或状态"))
		return
	}
	if req.DisplayName != nil {
		name := strings.TrimSpace(*req.DisplayName)
		if name == "" || len([]rune(name)) > 40 {
			writeError(w, badRequest("请填写 1–40 个字符的称呼"))
			return
		}
		target.DisplayName = name
	}
	if req.Relation != nil {
		rel := strings.TrimSpace(*req.Relation)
		if len([]rune(rel)) > 40 {
			writeError(w, badRequest("关系描述过长"))
			return
		}
		if rel == "" {
			target.Relation = nil
		} else {
			target.Relation = &rel
		}
	}
	if req.AvatarKey != nil {
		key := strings.TrimSpace(*req.AvatarKey)
		if key == "" {
			target.AvatarKey = nil
		} else {
			target.AvatarKey = &key
		}
	}
	wasActiveAdmin := target.Role == "admin" && target.Status == "active"
	demoting := false
	deactivating := false
	if req.Role != nil {
		if *req.Role != "admin" && *req.Role != "member" {
			writeError(w, badRequest("角色只能是管理员或成员"))
			return
		}
		if target.Role == "admin" && *req.Role != "admin" {
			demoting = true
		}
		target.Role = *req.Role
	}
	if req.Status != nil {
		if *req.Status != "active" && *req.Status != "inactive" {
			writeError(w, badRequest("状态只能是 active 或 inactive"))
			return
		}
		if isSelf && *req.Status == "inactive" {
			writeError(w, badRequest("不能停用自己的账号"))
			return
		}
		if target.Status == "active" && *req.Status == "inactive" {
			deactivating = true
		}
		target.Status = *req.Status
	}
	if wasActiveAdmin && (demoting || deactivating) {
		n, err := s.store.CountActiveAdmins(r.Context(), target.ID)
		if err != nil {
			writeError(w, serverError(err))
			return
		}
		if n == 0 {
			writeError(w, conflict("家庭至少需要保留一名启用中的管理员"))
			return
		}
	}
	if err := s.store.UpdateUser(r.Context(), target); err != nil {
		writeError(w, serverError(err))
		return
	}
	if target.Status == "inactive" {
		_ = s.store.DeleteUserSessions(r.Context(), target.ID)
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": target.Public()})
}

type resetPasswordRequest struct {
	Password string `json:"password"`
}

func (s *Server) handleMemberResetPassword(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	target, err := s.store.GetUserByID(r.Context(), id)
	if err != nil {
		writeError(w, notFound("成员不存在"))
		return
	}
	var req resetPasswordRequest
	if r.ContentLength != 0 {
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, err)
			return
		}
	}
	password := req.Password
	temporary := ""
	if password == "" {
		temporary, err = auth.GenerateTemporaryPassword()
		if err != nil {
			writeError(w, serverError(err))
			return
		}
		password = temporary
	} else if err := auth.ValidatePassword(password); err != nil {
		writeError(w, &APIError{Status: http.StatusBadRequest, Code: "auth.weak_password", Message: err.Error()})
		return
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	if err := s.store.UpdatePassword(r.Context(), target.ID, hash, true); err != nil {
		writeError(w, serverError(err))
		return
	}
	_ = s.store.DeleteUserSessions(r.Context(), target.ID)
	writeJSON(w, http.StatusOK, map[string]any{"temporaryPassword": temporary})
}

type householdPatchRequest struct {
	Name                   *string  `json:"name"`
	Timezone               *string  `json:"timezone"`
	WeekStart              *int     `json:"weekStart"`
	DateFormat             *string  `json:"dateFormat"`
	DefaultReminderOffset  *int     `json:"defaultReminderOffset"`
	DefaultWeatherCity     *string  `json:"defaultWeatherCity"`
	DefaultWeatherLat      *float64 `json:"defaultWeatherLat"`
	DefaultWeatherLon      *float64 `json:"defaultWeatherLon"`
	DefaultWeatherProvider *string  `json:"defaultWeatherProvider"`
	DefaultDashboard       *string  `json:"defaultDashboard"`
	Style                  *string  `json:"style"`
}

func (s *Server) handleHousehold(w http.ResponseWriter, r *http.Request) {
	h, err := s.store.GetHousehold(r.Context())
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"household": h})
}

func (s *Server) handleHouseholdPatch(w http.ResponseWriter, r *http.Request) {
	h, err := s.store.GetHousehold(r.Context())
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	var req householdPatchRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" || len([]rune(name)) > 40 {
			writeError(w, badRequest("家庭名称需为 1–40 个字符"))
			return
		}
		h.Name = name
	}
	if req.Timezone != nil {
		if _, err := time.LoadLocation(*req.Timezone); err != nil {
			writeError(w, badRequest("时区无效，请使用 IANA 时区名称"))
			return
		}
		h.Timezone = *req.Timezone
	}
	if req.WeekStart != nil {
		if *req.WeekStart < 0 || *req.WeekStart > 6 {
			writeError(w, badRequest("周起始日需在 0–6 之间"))
			return
		}
		h.WeekStart = *req.WeekStart
	}
	if req.DateFormat != nil {
		if *req.DateFormat != "24h" && *req.DateFormat != "12h" {
			writeError(w, badRequest("时间格式需为 24h 或 12h"))
			return
		}
		h.DateFormat = *req.DateFormat
	}
	if req.DefaultReminderOffset != nil {
		if *req.DefaultReminderOffset < 0 || *req.DefaultReminderOffset > 10080 {
			writeError(w, badRequest("默认提醒偏移需在 0–10080 分钟之间"))
			return
		}
		h.DefaultReminderOffset = *req.DefaultReminderOffset
	}
	if req.DefaultWeatherCity != nil {
		city := strings.TrimSpace(*req.DefaultWeatherCity)
		if city == "" {
			h.DefaultWeatherCity = nil
		} else {
			h.DefaultWeatherCity = &city
		}
	}
	if req.DefaultWeatherLat != nil {
		h.DefaultWeatherLat = req.DefaultWeatherLat
	}
	if req.DefaultWeatherLon != nil {
		h.DefaultWeatherLon = req.DefaultWeatherLon
	}
	if req.DefaultWeatherProvider != nil {
		provider := strings.TrimSpace(*req.DefaultWeatherProvider)
		if provider == "" {
			h.DefaultWeatherProvider = nil
		} else if !s.weather.Has(provider) {
			writeError(w, badRequest("未知的天气信息源"))
			return
		} else if !s.weather.Available(provider) {
			writeError(w, badRequest("该天气信息源尚未配置，请联系部署者"))
			return
		} else {
			h.DefaultWeatherProvider = &provider
		}
	}
	if req.DefaultDashboard != nil {
		if err := validateJSONField(*req.DefaultDashboard); err != nil {
			writeError(w, badRequest("卡片布局格式不正确"))
			return
		}
		h.DefaultDashboard = req.DefaultDashboard
	}
	if req.Style != nil {
		style := strings.TrimSpace(*req.Style)
		if style == "" {
			h.Style = nil
		} else {
			if err := validateJSONField(style); err != nil {
				writeError(w, badRequest("样式格式不正确"))
				return
			}
			h.Style = &style
		}
	}
	if err := s.store.UpdateHousehold(r.Context(), h); err != nil {
		writeError(w, serverError(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"household": h})
}

type memberStyleRequest struct {
	Dashboard *string `json:"dashboard"`
}

// handleMemberStyleGet 返回指定成员当前的首页卡片布局，供管理员调整。
func (s *Server) handleMemberStyleGet(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	if _, err := s.store.GetUserByID(r.Context(), id); err != nil {
		writeError(w, notFound("成员不存在"))
		return
	}
	layout, source, err := s.dashboardLayoutFor(r.Context(), id)
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"layout": layout, "layoutSource": source})
}

// handleMemberStylePatch 管理员为成员设定显示的卡片。
func (s *Server) handleMemberStylePatch(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	if _, err := s.store.GetUserByID(r.Context(), id); err != nil {
		writeError(w, notFound("成员不存在"))
		return
	}
	var req memberStyleRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	if req.Dashboard == nil {
		writeError(w, badRequest("请提供卡片布局"))
		return
	}
	prefs, err := s.store.GetPreferences(r.Context(), id)
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	raw := strings.TrimSpace(*req.Dashboard)
	if raw == "" {
		prefs.Dashboard = nil
	} else {
		var items []dashboardLayoutItem
		if err := json.Unmarshal([]byte(raw), &items); err != nil {
			writeError(w, badRequest("卡片布局格式不正确"))
			return
		}
		normalized, err := normalizeLayout(items)
		if err != nil {
			writeError(w, badRequest(err.Error()))
			return
		}
		encoded, err := encodeLayout(normalized)
		if err != nil {
			writeError(w, serverError(err))
			return
		}
		prefs.Dashboard = &encoded
	}
	if err := s.store.UpdatePreferences(r.Context(), prefs); err != nil {
		writeError(w, serverError(err))
		return
	}
	layout, source, err := s.dashboardLayoutFor(r.Context(), id)
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"layout": layout, "layoutSource": source})
}
