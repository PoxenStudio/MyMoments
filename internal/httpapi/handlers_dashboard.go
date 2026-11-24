package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"mymoments/internal/model"
	"mymoments/internal/store"
)

type dashboardCardDef struct {
	ID                string
	Title             string
	DefaultVisible    bool
	DefaultSize       string
	DefaultBackground string
}

// dashboardCardDefs 是卡片注册顺序与默认配置（PRD §6.2）。
var dashboardCardDefs = []dashboardCardDef{
	{"overview", "今日概览", true, "wide", "default"},
	{"tasks", "待办", true, "standard", "default"},
	{"calendar", "行事历", true, "standard", "default"},
	{"birthdays", "生日", true, "standard", "default"},
	{"reminders", "我的提醒", true, "standard", "default"},
	{"timeline", "家庭时光轴", false, "wide", "default"},
	{"messages", "留言", false, "standard", "default"},
	{"notes", "便签", false, "standard", "default"},
	{"weather", "天气", false, "standard", "default"},
	{"members", "家庭成员", false, "standard", "default"},
	{"contacts", "通讯录", false, "standard", "default"},
}

var dashboardSizes = map[string]bool{"standard": true, "wide": true, "tall": true}
var dashboardBackgrounds = map[string]bool{"default": true, "warm": true, "sage": true, "sky": true, "sand": true, "gradient": true}

type dashboardLayoutItem struct {
	ID         string `json:"id"`
	Visible    bool   `json:"visible"`
	Size       string `json:"size"`
	Background string `json:"background"`
}

type dashboardCard struct {
	ID         string  `json:"id"`
	Title      string  `json:"title"`
	Visible    bool    `json:"visible"`
	Size       string  `json:"size"`
	Background string  `json:"background"`
	Error      *string `json:"error"`
	Data       any     `json:"data"`
}

func dashboardCardTitle(id string) string {
	for _, def := range dashboardCardDefs {
		if def.ID == id {
			return def.Title
		}
	}
	return id
}

func defaultDashboardLayout() []dashboardLayoutItem {
	out := make([]dashboardLayoutItem, 0, len(dashboardCardDefs))
	for _, def := range dashboardCardDefs {
		out = append(out, dashboardLayoutItem{
			ID: def.ID, Visible: def.DefaultVisible, Size: def.DefaultSize, Background: def.DefaultBackground,
		})
	}
	return out
}

// normalizeLayout 校验并补全布局：未知/重复卡片报错，缺失卡片按默认值补在末尾。
func normalizeLayout(in []dashboardLayoutItem) ([]dashboardLayoutItem, error) {
	seen := map[string]bool{}
	out := make([]dashboardLayoutItem, 0, len(dashboardCardDefs))
	for _, item := range in {
		id := strings.TrimSpace(item.ID)
		if id == "" {
			return nil, fmt.Errorf("卡片标识不能为空")
		}
		known := false
		for _, def := range dashboardCardDefs {
			if def.ID == id {
				known = true
				break
			}
		}
		if !known {
			return nil, fmt.Errorf("未知的卡片：%s", id)
		}
		if seen[id] {
			return nil, fmt.Errorf("卡片重复：%s", id)
		}
		seen[id] = true
		size := item.Size
		if size == "" {
			size = "standard"
		}
		if !dashboardSizes[size] {
			return nil, fmt.Errorf("卡片尺寸无效：%s", size)
		}
		bg := item.Background
		if bg == "" {
			bg = "default"
		}
		if !dashboardBackgrounds[bg] {
			return nil, fmt.Errorf("卡片背景无效：%s", bg)
		}
		out = append(out, dashboardLayoutItem{ID: id, Visible: item.Visible, Size: size, Background: bg})
	}
	for _, def := range dashboardCardDefs {
		if !seen[def.ID] {
			out = append(out, dashboardLayoutItem{
				ID: def.ID, Visible: false, Size: def.DefaultSize, Background: def.DefaultBackground,
			})
		}
	}
	// 今日概览固定置顶且可见。
	for i, item := range out {
		if item.ID == "overview" {
			out[0], out[i] = out[i], out[0]
			out[0].Visible = true
			break
		}
	}
	return out, nil
}

func parseLayout(raw *string) ([]dashboardLayoutItem, error) {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return defaultDashboardLayout(), nil
	}
	var items []dashboardLayoutItem
	if err := json.Unmarshal([]byte(*raw), &items); err != nil {
		return defaultDashboardLayout(), nil
	}
	normalized, err := normalizeLayout(items)
	if err != nil {
		return defaultDashboardLayout(), nil
	}
	return normalized, nil
}

func encodeLayout(items []dashboardLayoutItem) (string, error) {
	b, err := json.Marshal(items)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// dashboardLayoutFor 返回当前用户生效的布局及其来源。
func (s *Server) dashboardLayoutFor(ctx context.Context, userID int64) ([]dashboardLayoutItem, string, error) {
	prefs, err := s.store.GetPreferences(ctx, userID)
	if err != nil {
		return nil, "", err
	}
	if prefs.Dashboard != nil && strings.TrimSpace(*prefs.Dashboard) != "" {
		items, err := parseLayout(prefs.Dashboard)
		if err == nil {
			return items, "personal", nil
		}
	}
	h, err := s.store.GetHousehold(ctx)
	if err != nil {
		return nil, "", err
	}
	if h.DefaultDashboard != nil && strings.TrimSpace(*h.DefaultDashboard) != "" {
		items, err := parseLayout(h.DefaultDashboard)
		if err == nil {
			return items, "household", nil
		}
	}
	return defaultDashboardLayout(), "default", nil
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	u, _ := currentUser(r)
	layout, source, err := s.dashboardLayoutFor(r.Context(), u.ID)
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	today := s.today()
	cards := make([]dashboardCard, 0, len(layout))
	for _, item := range layout {
		card := dashboardCard{
			ID: item.ID, Title: dashboardCardTitle(item.ID), Visible: item.Visible,
			Size: item.Size, Background: item.Background,
		}
		if item.Visible {
			data, err := s.safeCardData(r, u, item.ID, today)
			if err != nil {
				msg := err.Error()
				card.Error = &msg
			} else {
				card.Data = data
			}
		}
		cards = append(cards, card)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"layout": layout, "layoutSource": source, "cards": cards, "today": today,
	})
}

func (s *Server) handleDashboardLayout(w http.ResponseWriter, r *http.Request) {
	u, _ := currentUser(r)
	layout, source, err := s.dashboardLayoutFor(r.Context(), u.ID)
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"layout": layout, "layoutSource": source})
}

type layoutRequest struct {
	Layout []dashboardLayoutItem `json:"layout"`
	Reset  bool                  `json:"reset"`
}

func (s *Server) handleDashboardLayoutPut(w http.ResponseWriter, r *http.Request) {
	u, _ := currentUser(r)
	var req layoutRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	prefs, err := s.store.GetPreferences(r.Context(), u.ID)
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	if req.Reset {
		prefs.Dashboard = nil
	} else {
		if len(req.Layout) == 0 {
			writeError(w, badRequest("布局不能为空"))
			return
		}
		layout, err := normalizeLayout(req.Layout)
		if err != nil {
			writeError(w, badRequest(err.Error()))
			return
		}
		encoded, err := encodeLayout(layout)
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
	layout, source, err := s.dashboardLayoutFor(r.Context(), u.ID)
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"layout": layout, "layoutSource": source})
}

func (s *Server) handleHouseholdDefaultLayout(w http.ResponseWriter, r *http.Request) {
	var req layoutRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	h, err := s.store.GetHousehold(r.Context())
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	var layout []dashboardLayoutItem
	if req.Reset {
		h.DefaultDashboard = nil
		layout = defaultDashboardLayout()
	} else {
		if len(req.Layout) == 0 {
			writeError(w, badRequest("布局不能为空"))
			return
		}
		layout, err = normalizeLayout(req.Layout)
		if err != nil {
			writeError(w, badRequest(err.Error()))
			return
		}
		encoded, err := encodeLayout(layout)
		if err != nil {
			writeError(w, serverError(err))
			return
		}
		h.DefaultDashboard = &encoded
	}
	if err := s.store.UpdateHousehold(r.Context(), h); err != nil {
		writeError(w, serverError(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"layout": layout, "layoutSource": "household"})
}

// safeCardData 隔离单卡片失败，避免整页空白。
func (s *Server) safeCardData(r *http.Request, u *model.User, id, today string) (data any, err error) {
	defer func() {
		if rec := recover(); rec != nil {
			data = nil
			err = fmt.Errorf("内容加载失败")
		}
	}()
	if id == "weather" {
		lat, lon, city, apiErr := s.weatherLocationFor(r.Context(), u.ID, "", "", "")
		if apiErr != nil {
			return nil, fmt.Errorf("尚未设置天气位置")
		}
		weatherData, wErr := s.weather.Current(r.Context(), lat, lon, city)
		if wErr != nil {
			return nil, fmt.Errorf("天气服务暂时不可用")
		}
		return weatherData, nil
	}
	return s.cardData(r, u, id, today)
}

func (s *Server) cardData(r *http.Request, u *model.User, id, today string) (any, error) {
	ctx := r.Context()
	switch id {
	case "overview":
		return s.overviewCardData(ctx, u, today)
	case "tasks":
		counts, err := s.store.CountTasks(ctx, u.ID, today)
		if err != nil {
			return nil, err
		}
		items, err := s.store.ListTasks(ctx, store.TaskFilter{ViewerID: u.ID, Scope: "open", Limit: 6})
		if err != nil {
			return nil, err
		}
		return map[string]any{"items": items, "counts": counts}, nil
	case "calendar":
		events, err := s.expandedEvents(ctx, u.ID, today, datePlusDays(today, 7))
		if err != nil {
			return nil, err
		}
		if len(events) > 6 {
			events = events[:6]
		}
		return map[string]any{"items": events, "from": today, "to": datePlusDays(today, 7)}, nil
	case "timeline":
		return s.timelineCardData(r, u, today)
	case "birthdays":
		items, err := s.store.ListBirthdays(ctx, store.BirthdayFilter{})
		if err != nil {
			return nil, err
		}
		loc, err := s.location(ctx)
		if err != nil {
			return nil, err
		}
		now := time.Now().In(loc)
		for i := range items {
			decorateBirthday(&items[i], now)
		}
		sort.SliceStable(items, func(i, j int) bool {
			a, b := 9999, 9999
			if items[i].DaysUntil != nil {
				a = *items[i].DaysUntil
			}
			if items[j].DaysUntil != nil {
				b = *items[j].DaysUntil
			}
			if a == b {
				return items[i].Name < items[j].Name
			}
			return a < b
		})
		if len(items) > 5 {
			items = items[:5]
		}
		return map[string]any{"items": items}, nil
	case "reminders":
		items, err := s.store.ListReminders(ctx, u.ID, "pending")
		if err != nil {
			return nil, err
		}
		if len(items) > 5 {
			items = items[:5]
		}
		return map[string]any{"items": items}, nil
	case "messages":
		items, err := s.store.ListMessages(ctx, store.MessageFilter{ViewerID: u.ID, Limit: 3})
		if err != nil {
			return nil, err
		}
		return map[string]any{"items": items}, nil
	case "notes":
		items, err := s.store.ListNotes(ctx, store.NoteFilter{ViewerID: u.ID, Limit: 4})
		if err != nil {
			return nil, err
		}
		return map[string]any{"items": items}, nil
	case "members":
		users, err := s.store.ListUsers(ctx)
		if err != nil {
			return nil, err
		}
		items := make([]model.User, 0, len(users))
		for _, item := range users {
			if item.Status == "active" {
				items = append(items, item.Public())
			}
		}
		return map[string]any{"items": items}, nil
	case "contacts":
		items, err := s.store.ListContacts(ctx, store.ContactFilter{ViewerID: u.ID})
		if err != nil {
			return nil, err
		}
		if len(items) > 5 {
			items = items[:5]
		}
		return map[string]any{"items": items}, nil
	default:
		return map[string]any{}, nil
	}
}

func (s *Server) overviewCardData(ctx context.Context, u *model.User, today string) (any, error) {
	counts, err := s.store.CountTasks(ctx, u.ID, today)
	if err != nil {
		return nil, err
	}
	events, err := s.expandedEvents(ctx, u.ID, today, datePlusDays(today, 7))
	if err != nil {
		return nil, err
	}
	var nextEvent *model.CalendarEvent
	eventsToday := 0
	for i := range events {
		date := events[i].StartLocal
		if len(date) >= 10 {
			date = date[:10]
		}
		if events[i].OccurrenceDate != nil {
			date = *events[i].OccurrenceDate
		}
		if date == today {
			eventsToday++
		}
		if nextEvent == nil && date >= today {
			nextEvent = &events[i]
		}
	}
	birthdays, err := s.store.ListBirthdays(ctx, store.BirthdayFilter{})
	if err != nil {
		return nil, err
	}
	loc, err := s.location(ctx)
	if err != nil {
		return nil, err
	}
	now := time.Now().In(loc)
	upcoming := make([]model.Birthday, 0, 3)
	for i := range birthdays {
		decorateBirthday(&birthdays[i], now)
		if birthdays[i].DaysUntil != nil && *birthdays[i].DaysUntil <= 30 {
			upcoming = append(upcoming, birthdays[i])
		}
	}
	sort.SliceStable(upcoming, func(i, j int) bool {
		return *upcoming[i].DaysUntil < *upcoming[j].DaysUntil
	})
	if len(upcoming) > 3 {
		upcoming = upcoming[:3]
	}
	reminders, err := s.store.ListReminders(ctx, u.ID, "pending")
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"counts":           counts,
		"eventsToday":      eventsToday,
		"nextEvent":        nextEvent,
		"birthdays":        upcoming,
		"remindersPending": len(reminders),
	}, nil
}

func (s *Server) timelineCardData(r *http.Request, u *model.User, today string) (any, error) {
	loc, err := s.location(r.Context())
	if err != nil {
		return nil, err
	}
	from := datePlusDays(today, -30)
	types := parseTimelineTypes("")
	entries, _, err := s.collectTimeline(r, u, "past", today, from, today, types, "", loc)
	if err != nil {
		return nil, err
	}
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].OccurredOn == entries[j].OccurredOn {
			ai, aj := int64(0), int64(0)
			if entries[i].OccurredAt != nil {
				ai = *entries[i].OccurredAt
			}
			if entries[j].OccurredAt != nil {
				aj = *entries[j].OccurredAt
			}
			if ai == aj {
				return entries[i].OccurrenceKey < entries[j].OccurrenceKey
			}
			return ai > aj
		}
		return entries[i].OccurredOn > entries[j].OccurredOn
	})
	if len(entries) > 6 {
		entries = entries[:6]
	}
	return map[string]any{"items": entries}, nil
}

func (s *Server) expandedEvents(ctx context.Context, viewerID int64, from, to string) ([]model.CalendarEvent, error) {
	bases, err := s.store.ListEvents(ctx, viewerID, from, to, false)
	if err != nil {
		return nil, err
	}
	seriesIDs := make([]string, 0)
	for _, ev := range bases {
		if ev.SeriesID != nil && ev.RecurrenceRule != nil {
			seriesIDs = append(seriesIDs, *ev.SeriesID)
		}
	}
	exceptions, err := s.store.LoadEventExceptions(ctx, seriesIDs)
	if err != nil {
		return nil, err
	}
	items := make([]model.CalendarEvent, 0, len(bases))
	for _, ev := range bases {
		items = append(items, expandEvent(ev, from, to, exceptions)...)
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].StartLocal == items[j].StartLocal {
			return items[i].ID < items[j].ID
		}
		return items[i].StartLocal < items[j].StartLocal
	})
	return items, nil
}

func datePlusDays(date string, days int) string {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return date
	}
	return t.AddDate(0, 0, days).Format("2006-01-02")
}
