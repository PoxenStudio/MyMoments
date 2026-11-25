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

// 首页栅格：12 列、固定行高，与前端保持一致（参考 react-grid-layout / GridStack 的通用做法）。
const (
	dashboardColumns = 12
	dashboardMinW    = 3
	dashboardMinH    = 5
	dashboardMaxH    = 60
	dashboardMaxRows = 200
)

// dashboardCardDims 返回旧版尺寸对应的栅格宽高，用于旧布局迁移与默认值。
func dashboardCardDims(size string) (int, int) {
	switch size {
	case "wide":
		return 8, 6
	case "tall":
		return 4, 9
	default:
		return 4, 6
	}
}

type dashboardLayoutItem struct {
	ID         string `json:"id"`
	Visible    bool   `json:"visible"`
	Background string `json:"background"`
	X          int    `json:"x"`
	Y          int    `json:"y"`
	W          int    `json:"w"`
	H          int    `json:"h"`
	// Size 仅用于兼容旧版布局，新布局不再返回该字段。
	Size string `json:"size,omitempty"`
}

type dashboardCard struct {
	ID         string  `json:"id"`
	Title      string  `json:"title"`
	Visible    bool    `json:"visible"`
	Background string  `json:"background"`
	X          int     `json:"x"`
	Y          int     `json:"y"`
	W          int     `json:"w"`
	H          int     `json:"h"`
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

// defaultDashboardLayout 按 12 列栅格从左到右、从上到下紧凑排列默认卡片。
func defaultDashboardLayout() []dashboardLayoutItem {
	out := make([]dashboardLayoutItem, 0, len(dashboardCardDefs))
	x, y, rowHeight := 0, 0, 0
	for _, def := range dashboardCardDefs {
		w, h := dashboardCardDims(def.DefaultSize)
		if x+w > dashboardColumns {
			x = 0
			y += rowHeight
			rowHeight = 0
		}
		out = append(out, dashboardLayoutItem{
			ID: def.ID, Visible: def.DefaultVisible, Background: def.DefaultBackground,
			X: x, Y: y, W: w, H: h,
		})
		x += w
		if h > rowHeight {
			rowHeight = h
		}
	}
	return out
}

func normalizeBackground(raw string) (string, error) {
	if raw == "" {
		return "default", nil
	}
	if !dashboardBackgrounds[raw] {
		return "", fmt.Errorf("卡片背景无效：%s", raw)
	}
	return raw, nil
}

// normalizeRects 将卡片限制在栅格内，并保证最小宽高。
func normalizeRects(items []dashboardLayoutItem) {
	for i := range items {
		item := &items[i]
		if item.X < 0 {
			item.X = 0
		}
		if item.X > dashboardColumns-1 {
			item.X = dashboardColumns - 1
		}
		if item.W < dashboardMinW {
			item.W = dashboardMinW
		}
		if item.W > dashboardColumns {
			item.W = dashboardColumns
		}
		if item.X+item.W > dashboardColumns {
			item.X = dashboardColumns - item.W
			if item.X < 0 {
				item.X = 0
				item.W = dashboardColumns
			}
		}
		if item.H < dashboardMinH {
			item.H = dashboardMinH
		}
		if item.H > dashboardMaxH {
			item.H = dashboardMaxH
		}
		if item.Y < 0 {
			item.Y = 0
		}
		if item.Y+item.H > dashboardMaxRows {
			item.Y = dashboardMaxRows - item.H
			if item.Y < 0 {
				item.Y = 0
			}
		}
	}
}

func layoutItemsOverlap(a, b dashboardLayoutItem) bool {
	return a.X < b.X+b.W && a.X+a.W > b.X && a.Y < b.Y+b.H && a.Y+a.H > b.Y
}

// resolveLayoutOverlaps 保持数组顺序优先级，将冲突卡片依次下移，避免任意两卡重叠。
func resolveLayoutOverlaps(items []dashboardLayoutItem) {
	placed := make([]dashboardLayoutItem, 0, len(items))
	for i := range items {
		item := items[i]
		for {
			conflict := false
			for _, other := range placed {
				if layoutItemsOverlap(item, other) {
					conflict = true
					break
				}
			}
			if !conflict {
				break
			}
			item.Y++
		}
		items[i] = item
		placed = append(placed, item)
	}
}

// normalizeLayout 校验并补全布局：未知/重复卡片报错，缺失卡片按默认位置补在末尾。
// 旧版布局只包含 size 字段，此时按输入顺序重新排布；新版布局使用 x/y/w/h。
func normalizeLayout(in []dashboardLayoutItem) ([]dashboardLayoutItem, error) {
	if len(in) == 0 {
		return nil, fmt.Errorf("布局不能为空")
	}
	defs := defaultDashboardLayout()
	defByID := make(map[string]dashboardLayoutItem, len(defs))
	for _, def := range defs {
		defByID[def.ID] = def
	}

	seen := map[string]bool{}
	positioned := false
	cleaned := make([]dashboardLayoutItem, len(in))
	for i, item := range in {
		id := strings.TrimSpace(item.ID)
		if id == "" {
			return nil, fmt.Errorf("卡片标识不能为空")
		}
		if _, ok := defByID[id]; !ok {
			return nil, fmt.Errorf("未知的卡片：%s", id)
		}
		if seen[id] {
			return nil, fmt.Errorf("卡片重复：%s", id)
		}
		seen[id] = true
		if item.W > 0 || item.H > 0 {
			positioned = true
		}
		item.ID = id
		cleaned[i] = item
	}
	in = cleaned

	out := make([]dashboardLayoutItem, 0, len(defs))
	if !positioned {
		// 旧布局：沿用输入顺序与旧尺寸，重新贴合到栅格。
		x, y, rowHeight := 0, 0, 0
		place := func(item dashboardLayoutItem, w, h int) dashboardLayoutItem {
			if x+w > dashboardColumns {
				x = 0
				y += rowHeight
				rowHeight = 0
			}
			item.X, item.Y, item.W, item.H = x, y, w, h
			x += w
			if h > rowHeight {
				rowHeight = h
			}
			return item
		}
		for _, item := range in {
			base := defByID[item.ID]
			bg, err := normalizeBackground(item.Background)
			if err != nil {
				return nil, err
			}
			base.Visible = item.Visible
			base.Background = bg
			w, h := base.W, base.H
			if item.Size != "" {
				if !dashboardSizes[item.Size] {
					return nil, fmt.Errorf("卡片尺寸无效：%s", item.Size)
				}
				w, h = dashboardCardDims(item.Size)
			}
			out = append(out, place(base, w, h))
		}
		for _, def := range defs {
			if !seen[def.ID] {
				def.Visible = false
				out = append(out, place(def, def.W, def.H))
			}
		}
	} else {
		for _, item := range in {
			base := defByID[item.ID]
			bg, err := normalizeBackground(item.Background)
			if err != nil {
				return nil, err
			}
			w, h := item.W, item.H
			if w <= 0 || h <= 0 {
				dw, dh := base.W, base.H
				if item.Size != "" {
					if !dashboardSizes[item.Size] {
						return nil, fmt.Errorf("卡片尺寸无效：%s", item.Size)
					}
					dw, dh = dashboardCardDims(item.Size)
				}
				if w <= 0 {
					w = dw
				}
				if h <= 0 {
					h = dh
				}
			}
			x, y := item.X, item.Y
			if item.W <= 0 && item.H <= 0 {
				x, y = base.X, base.Y
			}
			out = append(out, dashboardLayoutItem{
				ID: item.ID, Visible: item.Visible, Background: bg,
				X: x, Y: y, W: w, H: h,
			})
		}
		for _, def := range defs {
			if !seen[def.ID] {
				def.Visible = false
				out = append(out, def)
			}
		}
	}

	normalizeRects(out)
	resolveLayoutOverlaps(out)
	// 今日概览始终可见。
	for i := range out {
		if out[i].ID == "overview" {
			out[i].Visible = true
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
			Background: item.Background, X: item.X, Y: item.Y, W: item.W, H: item.H,
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
		weatherData, _, wErr := s.weather.Current(r.Context(), s.weatherProvider(r.Context()), lat, lon, city)
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
		from, to := monthWindow(today)
		items, err := s.store.ListTasks(ctx, store.TaskFilter{ViewerID: u.ID, Scope: "open", From: from, To: to, Limit: 5})
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

// monthWindow 返回当前月与下一个月的首尾日期，用于首页待办卡筛选。
func monthWindow(date string) (string, string) {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return "", ""
	}
	first := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
	end := first.AddDate(0, 2, 0).AddDate(0, 0, -1)
	return first.Format("2006-01-02"), end.Format("2006-01-02")
}
