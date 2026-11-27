package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strings"
	"time"

	"mymoments/internal/auth"
	"mymoments/internal/model"
	"mymoments/internal/recur"
	"mymoments/internal/store"
)

type eventInput struct {
	Title           *string  `json:"title"`
	Description     *string  `json:"description"`
	StartLocal      *string  `json:"startLocal"`
	EndLocal        *string  `json:"endLocal"`
	AllDay          *bool    `json:"allDay"`
	Location        *string  `json:"location"`
	Color           *string  `json:"color"`
	Icon            *string  `json:"icon"`
	RecurrenceRule  *string  `json:"recurrenceRule"`
	Visibility      *string  `json:"visibility"`
	AssigneeIDs     *[]int64 `json:"assigneeIds"`
	ReminderOffsets *[]int   `json:"reminderOffsets"`
	Scope           *string  `json:"scope"`
	OccurrenceDate  *string  `json:"occurrenceDate"`
}

func (s *Server) handleEventsList(w http.ResponseWriter, r *http.Request) {
	u, _ := currentUser(r)
	q := r.URL.Query()
	from := q.Get("from")
	to := q.Get("to")
	today := s.today()
	if from == "" {
		from = today[:8] + "01"
	}
	if to == "" {
		to = monthEnd(from)
	}
	if err := recur.ValidateDate(from); err != nil {
		writeError(w, badRequest("开始日期格式需为 YYYY-MM-DD"))
		return
	}
	if err := recur.ValidateDate(to); err != nil {
		writeError(w, badRequest("结束日期格式需为 YYYY-MM-DD"))
		return
	}
	if to < from {
		writeError(w, badRequest("结束日期不能早于开始日期"))
		return
	}
	bases, err := s.store.ListEvents(r.Context(), u.ID, from, to, q.Get("source") == "all")
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	seriesIDs := make([]string, 0)
	for _, ev := range bases {
		if ev.SeriesID != nil && ev.RecurrenceRule != nil {
			seriesIDs = append(seriesIDs, *ev.SeriesID)
		}
	}
	exceptions, err := s.store.LoadEventExceptions(r.Context(), seriesIDs)
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	items := make([]model.CalendarEvent, 0)
	for _, ev := range bases {
		items = append(items, expandEvent(ev, from, to, exceptions)...)
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].StartLocal == items[j].StartLocal {
			return items[i].ID < items[j].ID
		}
		return items[i].StartLocal < items[j].StartLocal
	})
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "from": from, "to": to})
}

func expandEvent(ev model.CalendarEvent, from, to string, exceptions map[string]map[string]store.EventException) []model.CalendarEvent {
	anchorDate, _, _ := splitLocal(ev.StartLocal)
	out := []model.CalendarEvent{}
	if ev.RecurrenceRule == nil || ev.SeriesID == nil {
		date := anchorDate
		ev.OccurrenceDate = &date
		return append(out, ev)
	}
	rule, err := recur.Parse(*ev.RecurrenceRule)
	if err != nil || rule == nil {
		date := anchorDate
		ev.OccurrenceDate = &date
		return append(out, ev)
	}
	dates := recur.Expand(rule, anchorDate, from, to)
	exc := exceptions[*ev.SeriesID]
	for _, date := range dates {
		item := ev
		if e, ok := exc[date]; ok {
			if e.Action == "cancel" {
				continue
			}
			store.ParseOverride(e.OverrideFields).Apply(&item)
		}
		shiftOccurrence(&item, anchorDate, date)
		item.OccurrenceDate = &date
		out = append(out, item)
	}
	return out
}

func (s *Server) handleEventGet(w http.ResponseWriter, r *http.Request) {
	u, _ := currentUser(r)
	ev, ok := s.loadEventFor(r, w)
	if !ok {
		return
	}
	if !canReadEvent(u, ev) {
		writeError(w, notFound("事件不存在"))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"event": ev})
}

func (s *Server) handleEventCreate(w http.ResponseWriter, r *http.Request) {
	u, _ := currentUser(r)
	var in eventInput
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, err)
		return
	}
	ev := &model.CalendarEvent{
		Visibility:      "household",
		Source:          "manual",
		CreatedBy:       u.ID,
		Assignees:       []model.User{},
		ReminderOffsets: []int{},
	}
	if apiErr := s.applyEventInput(r, ev, &in, true); apiErr != nil {
		writeError(w, apiErr)
		return
	}
	id, err := s.store.CreateEvent(r.Context(), ev)
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	ev.ID = id
	if err := s.syncEventReminders(r, ev); err != nil {
		writeError(w, serverError(err))
		return
	}
	created, err := s.store.GetEvent(r.Context(), id)
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"event": created})
}

func (s *Server) handleEventPatch(w http.ResponseWriter, r *http.Request) {
	u, _ := currentUser(r)
	ev, ok := s.loadEventFor(r, w)
	if !ok {
		return
	}
	if !canReadEvent(u, ev) {
		writeError(w, notFound("事件不存在"))
		return
	}
	if !canWriteEvent(u, ev) {
		writeError(w, forbidden("没有修改该事件的权限"))
		return
	}
	if ev.Source == "birthday" {
		writeError(w, badRequest("生日事件请在生日管理中修改"))
		return
	}
	var in eventInput
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, err)
		return
	}
	scope := valueOr(in.Scope, "all")
	if ev.RecurrenceRule != nil && ev.SeriesID != nil && (scope == "this" || scope == "following") {
		occ := ""
		if in.OccurrenceDate != nil {
			occ = strings.TrimSpace(*in.OccurrenceDate)
		}
		if err := recur.ValidateDate(occ); err != nil {
			writeError(w, badRequest("请指定要修改的发生日期"))
			return
		}
		if scope == "this" {
			if err := s.overrideOccurrence(r, ev, occ, &in); err != nil {
				writeError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"event": ev, "scope": "this", "occurrenceDate": occ})
			return
		}
		newEvent, err := s.splitSeries(r, ev, occ, &in)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"event": newEvent, "scope": "following"})
		return
	}
	if apiErr := s.applyEventInput(r, ev, &in, false); apiErr != nil {
		writeError(w, apiErr)
		return
	}
	ev.Source = "manual"
	if ev.SourceRef != nil {
		ev.SourceRef = nil
	}
	if err := s.store.UpdateEvent(r.Context(), ev); err != nil {
		writeError(w, serverError(err))
		return
	}
	if err := s.syncEventReminders(r, ev); err != nil {
		writeError(w, serverError(err))
		return
	}
	updated, err := s.store.GetEvent(r.Context(), ev.ID)
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"event": updated})
}

func (s *Server) handleEventDelete(w http.ResponseWriter, r *http.Request) {
	u, _ := currentUser(r)
	ev, ok := s.loadEventFor(r, w)
	if !ok {
		return
	}
	if !canReadEvent(u, ev) {
		writeError(w, notFound("事件不存在"))
		return
	}
	if ev.Source == "birthday" {
		writeError(w, badRequest("生日事件请在生日管理中删除"))
		return
	}
	q := r.URL.Query()
	scope := q.Get("scope")
	if scope == "" {
		scope = "all"
	}
	occ := q.Get("occurrence")
	if ev.RecurrenceRule != nil && ev.SeriesID != nil && (scope == "this" || scope == "following") {
		if err := recur.ValidateDate(occ); err != nil {
			writeError(w, badRequest("请指定要删除的发生日期"))
			return
		}
		if !canDeleteEvent(u, ev) {
			writeError(w, forbidden("只有创建者或管理员可以删除该事件"))
			return
		}
		if scope == "this" {
			if err := s.store.UpsertEventException(r.Context(), store.EventException{
				SeriesID: *ev.SeriesID, OccurrenceDate: occ, Action: "cancel",
			}); err != nil {
				writeError(w, serverError(err))
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		anchorDate, _, _ := splitLocal(ev.StartLocal)
		if occ <= anchorDate {
			if err := s.store.DeleteEvent(r.Context(), ev.ID); err != nil {
				writeError(w, serverError(err))
				return
			}
			_ = s.store.DeleteSourceReminders(r.Context(), "event", ev.ID)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		prev, err := shiftDate(occ, -1)
		if err != nil {
			writeError(w, serverError(err))
			return
		}
		rule, _ := recur.Parse(*ev.RecurrenceRule)
		rule.Until = prev
		raw, _ := json.Marshal(rule)
		ruleText := string(raw)
		ev.RecurrenceRule = &ruleText
		if err := s.store.UpdateEvent(r.Context(), ev); err != nil {
			writeError(w, serverError(err))
			return
		}
		_ = s.store.DeleteEventExceptionsFrom(r.Context(), *ev.SeriesID, occ)
		_ = s.store.DeleteSourceReminders(r.Context(), "event", ev.ID)
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if !canDeleteEvent(u, ev) {
		writeError(w, forbidden("只有创建者或管理员可以删除该事件"))
		return
	}
	if err := s.store.DeleteEvent(r.Context(), ev.ID); err != nil {
		writeError(w, serverError(err))
		return
	}
	_ = s.store.DeleteSourceReminders(r.Context(), "event", ev.ID)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) loadEventFor(r *http.Request, w http.ResponseWriter) (*model.CalendarEvent, bool) {
	id, err := parseID(r, "id")
	if err != nil {
		writeError(w, err)
		return nil, false
	}
	ev, err := s.store.GetEvent(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, notFound("事件不存在"))
			return nil, false
		}
		writeError(w, serverError(err))
		return nil, false
	}
	return ev, true
}

func (s *Server) applyEventInput(r *http.Request, ev *model.CalendarEvent, in *eventInput, create bool) *APIError {
	if in.Title != nil {
		title := strings.TrimSpace(*in.Title)
		if title == "" || len([]rune(title)) > 120 {
			return badRequest("请填写 1–120 个字符的标题")
		}
		ev.Title = title
	} else if create {
		return badRequest("请填写事件标题")
	}
	if in.Description != nil {
		desc := strings.TrimSpace(*in.Description)
		if len([]rune(desc)) > 5000 {
			return badRequest("说明过长")
		}
		ev.Description = nonEmptyPtr(desc)
	}
	if in.AllDay != nil {
		ev.AllDay = *in.AllDay
	}
	if in.StartLocal != nil {
		start, err := normalizeLocal(*in.StartLocal, ev.AllDay)
		if err != nil {
			return badRequest("开始时间格式不正确")
		}
		ev.StartLocal = start
	} else if create {
		return badRequest("请选择事件开始时间")
	}
	if in.EndLocal != nil {
		if strings.TrimSpace(*in.EndLocal) == "" {
			ev.EndLocal = nil
		} else {
			end, err := normalizeLocal(*in.EndLocal, ev.AllDay)
			if err != nil {
				return badRequest("结束时间格式不正确")
			}
			ev.EndLocal = &end
		}
	}
	if ev.AllDay {
		if date, _, ok := splitLocal(ev.StartLocal); ok {
			ev.StartLocal = date + " 00:00:00"
		}
		if ev.EndLocal != nil {
			if date, _, ok := splitLocal(*ev.EndLocal); ok {
				end := date + " 23:59:59"
				ev.EndLocal = &end
			}
		}
	}
	if ev.EndLocal != nil && *ev.EndLocal < ev.StartLocal {
		return badRequest("结束时间不能早于开始时间")
	}
	if in.Location != nil {
		ev.Location = nonEmptyPtr(*in.Location)
	}
	if in.Color != nil {
		ev.Color = nonEmptyPtr(*in.Color)
	}
	if in.Icon != nil {
		ev.Icon = nonEmptyPtr(*in.Icon)
	}
	if in.Visibility != nil {
		if !validVisibility(*in.Visibility) {
			return badRequest("可见范围无效")
		}
		ev.Visibility = *in.Visibility
	}
	if in.RecurrenceRule != nil {
		raw := strings.TrimSpace(*in.RecurrenceRule)
		if raw == "" {
			ev.RecurrenceRule = nil
			ev.SeriesID = nil
		} else {
			if _, err := recur.Parse(raw); err != nil {
				return badRequest(err.Error())
			}
			ev.RecurrenceRule = &raw
			if ev.SeriesID == nil {
				series, err := auth.RandomToken(12)
				if err != nil {
					return serverError(err)
				}
				ev.SeriesID = &series
			}
		}
	}
	if in.AssigneeIDs != nil {
		users, err := s.resolveAssignees(r, *in.AssigneeIDs)
		if err != nil {
			return asAPIError(err)
		}
		ev.Assignees = users
	}
	if in.ReminderOffsets != nil {
		ev.ReminderOffsets = normalizeOffsets(*in.ReminderOffsets)
	}
	date, _, _ := splitLocal(ev.StartLocal)
	ev.OccurrenceDate = &date
	return nil
}

func (s *Server) overrideOccurrence(r *http.Request, ev *model.CalendarEvent, occ string, in *eventInput) error {
	ov := store.EventOccurrenceOverride{}
	if in.Title != nil {
		title := strings.TrimSpace(*in.Title)
		if title == "" || len([]rune(title)) > 120 {
			return badRequest("请填写 1–120 个字符的标题")
		}
		ov.Title = &title
	}
	if in.Description != nil {
		ov.Description = in.Description
	}
	if in.StartLocal != nil {
		start, err := normalizeLocal(*in.StartLocal, ev.AllDay)
		if err != nil {
			return badRequest("开始时间格式不正确")
		}
		ov.StartLocal = &start
	}
	if in.EndLocal != nil {
		end, err := normalizeLocal(*in.EndLocal, ev.AllDay)
		if err != nil {
			return badRequest("结束时间格式不正确")
		}
		ov.EndLocal = &end
	}
	if in.AllDay != nil {
		ov.AllDay = in.AllDay
	}
	if in.Location != nil {
		ov.Location = in.Location
	}
	if in.Color != nil {
		ov.Color = in.Color
	}
	if in.Icon != nil {
		ov.Icon = in.Icon
	}
	raw, _ := json.Marshal(ov)
	return s.store.UpsertEventException(r.Context(), store.EventException{
		SeriesID: *ev.SeriesID, OccurrenceDate: occ, Action: "override", OverrideFields: string(raw),
	})
}

func (s *Server) splitSeries(r *http.Request, ev *model.CalendarEvent, occ string, in *eventInput) (*model.CalendarEvent, error) {
	anchorDate, _, _ := splitLocal(ev.StartLocal)
	updates := *ev
	updates.ID = 0
	updates.CreatedBy = ev.CreatedBy
	updates.Source = "manual"
	updates.SourceRef = nil
	updates.SeriesID = nil
	updates.Assignees = append([]model.User{}, ev.Assignees...)
	updates.ReminderOffsets = append([]int{}, ev.ReminderOffsets...)
	days := daysBetween(anchorDate, occ)
	shifted, err := shiftDate(ev.StartLocal[:10], days)
	if err != nil {
		return nil, serverError(err)
	}
	updates.StartLocal = shifted + ev.StartLocal[10:]
	if ev.EndLocal != nil {
		endShifted, err := shiftDate((*ev.EndLocal)[:10], days)
		if err != nil {
			return nil, serverError(err)
		}
		end := endShifted + (*ev.EndLocal)[10:]
		updates.EndLocal = &end
	}
	if apiErr := s.applyEventInput(r, &updates, in, false); apiErr != nil {
		return nil, apiErr
	}
	series, err := auth.RandomToken(12)
	if err != nil {
		return nil, serverError(err)
	}
	updates.SeriesID = &series
	updates.ID = 0
	newID, err := s.store.CreateEvent(r.Context(), &updates)
	if err != nil {
		return nil, serverError(err)
	}
	updates.ID = newID
	if ev.SeriesID != nil {
		if occ <= anchorDate {
			if err := s.store.DeleteEvent(r.Context(), ev.ID); err != nil {
				return nil, serverError(err)
			}
			_ = s.store.DeleteSourceReminders(r.Context(), "event", ev.ID)
		} else {
			prev, err := shiftDate(occ, -1)
			if err != nil {
				return nil, serverError(err)
			}
			rule, _ := recur.Parse(*ev.RecurrenceRule)
			rule.Until = prev
			raw, _ := json.Marshal(rule)
			text := string(raw)
			ev.RecurrenceRule = &text
			if err := s.store.UpdateEvent(r.Context(), ev); err != nil {
				return nil, serverError(err)
			}
			_ = s.store.DeleteEventExceptionsFrom(r.Context(), *ev.SeriesID, occ)
		}
	}
	if err := s.syncEventReminders(r, &updates); err != nil {
		return nil, serverError(err)
	}
	created, err := s.store.GetEvent(r.Context(), newID)
	if err != nil {
		return nil, serverError(err)
	}
	return created, nil
}

func (s *Server) syncEventReminders(r *http.Request, ev *model.CalendarEvent) error {
	if len(ev.ReminderOffsets) == 0 {
		return s.store.ReplaceSourceReminders(r.Context(), "event", ev.ID, nil)
	}
	loc, err := s.location(r.Context())
	if err != nil {
		return err
	}
	anchorDate, startClock, _ := splitLocal(ev.StartLocal)
	if ev.AllDay {
		startClock = "09:00:00"
	}
	dates := []string{anchorDate}
	// 非重复事件没有 recurrenceRule，必须判空，否则带提醒的单次事件会 panic。
	if ev.RecurrenceRule != nil && ev.SeriesID != nil {
		if rule, err := recur.Parse(*ev.RecurrenceRule); err == nil && rule != nil {
			from := s.today()
			to := time.Now().In(loc).AddDate(2, 0, 0).Format("2006-01-02")
			dates = recur.Expand(rule, anchorDate, from, to)
		}
	}
	recipients := make([]int64, 0, len(ev.Assignees))
	for _, a := range ev.Assignees {
		recipients = append(recipients, a.ID)
	}
	if len(recipients) == 0 {
		recipients = []int64{ev.CreatedBy}
	}
	specs := []store.ReminderSpec{}
	for _, date := range dates {
		start, err := time.ParseInLocation("2006-01-02 15:04:05", date+" "+startClock, loc)
		if err != nil {
			continue
		}
		occ := date
		for _, uid := range recipients {
			for _, offset := range ev.ReminderOffsets {
				specs = append(specs, store.ReminderSpec{
					UserID:         uid,
					RemindAt:       start.Unix() - int64(offset)*60,
					OccurrenceDate: &occ,
					Title:          ev.Title,
				})
			}
		}
	}
	return s.store.ReplaceSourceReminders(r.Context(), "event", ev.ID, specs)
}

// normalizeLocal 将日期或日期时间规范为家庭时区的本地时间字符串。
func normalizeLocal(v string, allDay bool) (string, error) {
	v = strings.TrimSpace(strings.ReplaceAll(v, "T", " "))
	if v == "" {
		return "", errors.New("empty")
	}
	if len(v) == 10 {
		if err := recur.ValidateDate(v); err != nil {
			return "", err
		}
		if allDay {
			return v + " 00:00:00", nil
		}
		return v + " 00:00:00", nil
	}
	if len(v) == 16 {
		v += ":00"
	}
	parsed, err := time.Parse("2006-01-02 15:04:05", v)
	if err != nil {
		return "", err
	}
	return parsed.Format("2006-01-02 15:04:05"), nil
}

func splitLocal(local string) (string, string, bool) {
	parts := strings.SplitN(strings.TrimSpace(local), " ", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func shiftOccurrence(ev *model.CalendarEvent, anchorDate, occurrenceDate string) {
	days := daysBetween(anchorDate, occurrenceDate)
	if days == 0 {
		return
	}
	if d, err := shiftDate(ev.StartLocal[:10], days); err == nil {
		ev.StartLocal = d + ev.StartLocal[10:]
	}
	if ev.EndLocal != nil && len(*ev.EndLocal) >= 10 {
		if d, err := shiftDate((*ev.EndLocal)[:10], days); err == nil {
			end := d + (*ev.EndLocal)[10:]
			ev.EndLocal = &end
		}
	}
}

func monthEnd(date string) string {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return date
	}
	return t.AddDate(0, 1, -1).Format("2006-01-02")
}

func asAPIError(err error) *APIError {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr
	}
	return serverError(err)
}
