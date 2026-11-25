package httpapi

import (
	"context"
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"mymoments/internal/auth"
	"mymoments/internal/model"
	"mymoments/internal/store"
)

type birthdayInput struct {
	Name           *string `json:"name"`
	Month          *int    `json:"month"`
	Day            *int    `json:"day"`
	YearUnknown    *bool   `json:"yearUnknown"`
	BirthYear      *int    `json:"birthYear"`
	Note           *string `json:"note"`
	AvatarKey      *string `json:"avatarKey"`
	ContactID      *int64  `json:"contactId"`
	UserID         *int64  `json:"userId"`
	ReminderOffset *int    `json:"reminderOffset"`
}

func (s *Server) handleBirthdaysList(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	items, err := s.store.ListBirthdays(r.Context(), store.BirthdayFilter{Query: q.Get("q"), Source: q.Get("source")})
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	loc, err := s.location(r.Context())
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	now := time.Now().In(loc)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	for i := range items {
		decorateBirthday(&items[i], today)
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
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "today": today.Format("2006-01-02")})
}

func (s *Server) handleBirthdayCreate(w http.ResponseWriter, r *http.Request) {
	u, _ := currentUser(r)
	var in birthdayInput
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, err)
		return
	}
	b := &model.Birthday{CreatedBy: u.ID}
	if apiErr := s.applyBirthdayInput(r, b, &in, true); apiErr != nil {
		writeError(w, apiErr)
		return
	}
	id, err := s.store.CreateBirthday(r.Context(), b)
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	b.ID = id
	if err := s.syncBirthdayEvent(r.Context(), b); err != nil {
		writeError(w, serverError(err))
		return
	}
	if err := s.syncBirthdayReminder(r.Context(), b); err != nil {
		writeError(w, serverError(err))
		return
	}
	created, err := s.store.GetBirthday(r.Context(), id)
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	loc, _ := s.location(r.Context())
	decorateBirthday(created, time.Now().In(loc))
	writeJSON(w, http.StatusCreated, map[string]any{"birthday": created})
}

func (s *Server) handleBirthdayPatch(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	b, err := s.store.GetBirthday(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, notFound("生日不存在"))
			return
		}
		writeError(w, serverError(err))
		return
	}
	u, _ := currentUser(r)
	if b.CreatedBy != u.ID && u.Role != "admin" {
		writeError(w, forbidden("只有创建者或管理员可以修改该生日"))
		return
	}
	var in birthdayInput
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, err)
		return
	}
	if apiErr := s.applyBirthdayInput(r, b, &in, false); apiErr != nil {
		writeError(w, apiErr)
		return
	}
	if err := s.store.UpdateBirthday(r.Context(), b); err != nil {
		writeError(w, serverError(err))
		return
	}
	if err := s.syncBirthdayEvent(r.Context(), b); err != nil {
		writeError(w, serverError(err))
		return
	}
	if err := s.syncBirthdayReminder(r.Context(), b); err != nil {
		writeError(w, serverError(err))
		return
	}
	updated, err := s.store.GetBirthday(r.Context(), id)
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	loc, _ := s.location(r.Context())
	decorateBirthday(updated, time.Now().In(loc))
	writeJSON(w, http.StatusOK, map[string]any{"birthday": updated})
}

func (s *Server) handleBirthdayDelete(w http.ResponseWriter, r *http.Request) {
	u, _ := currentUser(r)
	id, err := parseID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	b, err := s.store.GetBirthday(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, notFound("生日不存在"))
			return
		}
		writeError(w, serverError(err))
		return
	}
	if b.CreatedBy != u.ID && u.Role != "admin" {
		writeError(w, forbidden("只有创建者或管理员可以删除该生日"))
		return
	}
	if b.CalendarEventID != nil {
		_ = s.store.DeleteEvent(r.Context(), *b.CalendarEventID)
		_ = s.store.DeleteSourceReminders(r.Context(), "event", *b.CalendarEventID)
	}
	if b.ContactID != nil {
		if c, err := s.store.GetContact(r.Context(), *b.ContactID); err == nil && c.ShowInCalendar {
			c.ShowInCalendar = false
			_ = s.store.UpdateContact(r.Context(), c)
		}
	}
	_ = s.store.DeleteSourceReminders(r.Context(), "birthday", b.ID)
	if err := s.store.DeleteBirthday(r.Context(), id); err != nil {
		writeError(w, serverError(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) applyBirthdayInput(r *http.Request, b *model.Birthday, in *birthdayInput, create bool) *APIError {
	if in.Name != nil {
		name := strings.TrimSpace(*in.Name)
		if name == "" || len([]rune(name)) > 80 {
			return badRequest("请填写 1–80 个字符的姓名")
		}
		b.Name = name
	} else if create {
		return badRequest("请填写姓名")
	}
	if in.Month != nil {
		if *in.Month < 1 || *in.Month > 12 {
			return badRequest("月份需在 1–12 之间")
		}
		b.Month = *in.Month
	}
	if in.Day != nil {
		b.Day = *in.Day
	}
	if create && (b.Month == 0 || b.Day == 0) {
		return badRequest("请填写生日月份和日期")
	}
	if b.Month > 0 && b.Day > 0 {
		last := time.Date(2024, time.Month(b.Month)+1, 0, 0, 0, 0, 0, time.UTC).Day()
		if b.Month == 2 {
			last = 29
		}
		if b.Day < 1 || b.Day > last {
			return badRequest("生日日期无效")
		}
	}
	if in.YearUnknown != nil {
		b.YearUnknown = *in.YearUnknown
	}
	if in.BirthYear != nil {
		if *in.BirthYear != 0 {
			if *in.BirthYear < 1900 || *in.BirthYear > time.Now().Year() {
				return badRequest("出生年份无效")
			}
			b.BirthYear = in.BirthYear
			b.YearUnknown = false
		} else {
			b.BirthYear = nil
			b.YearUnknown = true
		}
	}
	if b.YearUnknown {
		b.BirthYear = nil
	}
	if in.Note != nil {
		note := strings.TrimSpace(*in.Note)
		if len([]rune(note)) > 1000 {
			return badRequest("备注过长")
		}
		b.Note = nonEmptyPtr(note)
	}
	if in.AvatarKey != nil {
		b.AvatarKey = nonEmptyPtr(*in.AvatarKey)
	}
	if in.ContactID != nil {
		if *in.ContactID > 0 {
			if _, err := s.store.GetContact(r.Context(), *in.ContactID); err != nil {
				return badRequest("关联的联系人不存在")
			}
			b.ContactID = in.ContactID
		} else {
			b.ContactID = nil
		}
	}
	if in.UserID != nil {
		if *in.UserID > 0 {
			if _, err := s.store.GetUserByID(r.Context(), *in.UserID); err != nil {
				return badRequest("关联的家庭成员不存在")
			}
			existing, err := s.store.GetBirthdayByUser(r.Context(), *in.UserID)
			if err == nil && existing.ID != b.ID {
				return conflict("该成员已有生日记录")
			}
			b.UserID = in.UserID
		} else {
			b.UserID = nil
		}
	}
	if in.ReminderOffset != nil {
		if *in.ReminderOffset < 0 || *in.ReminderOffset > 10080 {
			return badRequest("提醒提前量无效")
		}
		offset := *in.ReminderOffset
		b.ReminderOffset = &offset
	}
	return nil
}

func decorateBirthday(b *model.Birthday, today time.Time) {
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())
	next := nextBirthdayDate(b.Month, b.Day, today)
	days := int(next.Sub(today).Hours() / 24)
	b.DaysUntil = &days
	if !b.YearUnknown && b.BirthYear != nil {
		age := next.Year() - *b.BirthYear
		if age < 0 {
			age = 0
		}
		b.Age = &age
	}
}

func nextBirthdayDate(month, day int, today time.Time) time.Time {
	for i := 0; i < 9; i++ {
		year := today.Year() + i
		last := time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, today.Location()).Day()
		d := day
		if d > last {
			d = last
		}
		candidate := time.Date(year, time.Month(month), d, 0, 0, 0, 0, today.Location())
		if !candidate.Before(today) {
			return candidate
		}
	}
	return today
}

func (s *Server) syncBirthdayEvent(ctx context.Context, b *model.Birthday) error {
	title := b.Name + "的生日"
	anchor := birthdayAnchorDate(b.Month, b.Day)
	rule := `{"freq":"yearly","interval":1,"anchor":"due"}`
	var ev *model.CalendarEvent
	if b.CalendarEventID != nil {
		ev, _ = s.store.GetEvent(ctx, *b.CalendarEventID)
	}
	if ev == nil {
		series, err := randomSeries()
		if err != nil {
			return err
		}
		ref := strconv.FormatInt(b.ID, 10)
		created := &model.CalendarEvent{
			Title:           title,
			StartLocal:      anchor + " 00:00:00",
			AllDay:          true,
			RecurrenceRule:  &rule,
			SeriesID:        &series,
			Visibility:      "household",
			Source:          "birthday",
			SourceRef:       &ref,
			CreatedBy:       b.CreatedBy,
			Assignees:       []model.User{},
			ReminderOffsets: []int{},
		}
		id, err := s.store.CreateEvent(ctx, created)
		if err != nil {
			return err
		}
		b.CalendarEventID = &id
		return s.store.UpdateBirthday(ctx, b)
	}
	ev.Title = title
	ev.StartLocal = anchor + " 00:00:00"
	ev.AllDay = true
	ev.RecurrenceRule = &rule
	ev.Visibility = "household"
	return s.store.UpdateEvent(ctx, ev)
}

func (s *Server) syncBirthdayReminder(ctx context.Context, b *model.Birthday) error {
	if b.ReminderOffset == nil {
		return s.store.ReplaceSourceReminders(ctx, "birthday", b.ID, nil)
	}
	loc, err := s.location(ctx)
	if err != nil {
		return err
	}
	now := time.Now().In(loc)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	next := nextBirthdayDate(b.Month, b.Day, today)
	at := time.Date(next.Year(), next.Month(), next.Day(), 9, 0, 0, 0, loc).Unix() - int64(*b.ReminderOffset)*60
	if at <= time.Now().Unix() {
		return nil
	}
	occ := next.Format("2006-01-02")
	return s.store.ReplaceSourceReminders(ctx, "birthday", b.ID, []store.ReminderSpec{{
		UserID:         b.CreatedBy,
		RemindAt:       at,
		OccurrenceDate: &occ,
		Title:          b.Name + "的生日",
	}})
}

func birthdayAnchorDate(month, day int) string {
	year := time.Now().Year()
	if month == 2 && day == 29 {
		for year%4 != 0 {
			year++
		}
	}
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
}

func randomSeries() (string, error) {
	return auth.RandomToken(12)
}
