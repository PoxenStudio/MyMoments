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
	"mymoments/internal/holiday"
	"mymoments/internal/model"
	"mymoments/internal/store"
)

type birthdayDateInput struct {
	Calendar       string `json:"calendar"`
	Month          *int   `json:"month"`
	Day            *int   `json:"day"`
	YearUnknown    *bool  `json:"yearUnknown"`
	BirthYear      *int   `json:"birthYear"`
	ReminderOffset *int   `json:"reminderOffset"`
}

type birthdayInput struct {
	Name      *string             `json:"name"`
	Note      *string             `json:"note"`
	AvatarKey *string             `json:"avatarKey"`
	ContactID *int64              `json:"contactId"`
	UserID    *int64              `json:"userId"`
	Dates     []birthdayDateInput `json:"dates"`
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
	from, to, hasRange, apiErr := parseBirthdayRange(q.Get("from"), q.Get("to"), loc)
	if apiErr != nil {
		writeError(w, apiErr)
		return
	}
	for i := range items {
		decorateBirthday(&items[i], today)
		if hasRange {
			items[i].Occurrences = birthdayOccurrencesInRange(&items[i], from, to)
		}
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
	for _, d := range b.Dates {
		if d.CalendarEventID != nil {
			_ = s.store.DeleteEvent(r.Context(), *d.CalendarEventID)
			_ = s.store.DeleteSourceReminders(r.Context(), "event", *d.CalendarEventID)
		}
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
	if create || in.Dates != nil {
		dates, apiErr := s.buildBirthdayDates(in.Dates, b, create)
		if apiErr != nil {
			return apiErr
		}
		b.Dates = dates
	}
	if !hasSolarDate(b.Dates) {
		return badRequest("请填写公历生日月份和日期")
	}
	return nil
}

func (s *Server) buildBirthdayDates(inputs []birthdayDateInput, b *model.Birthday, create bool) ([]model.BirthdayDate, *APIError) {
	if create && len(inputs) == 0 {
		return nil, badRequest("请填写公历生日月份和日期")
	}
	existingEvents := make(map[string]*int64, len(b.Dates))
	for i := range b.Dates {
		existingEvents[b.Dates[i].Calendar] = b.Dates[i].CalendarEventID
	}
	out := make([]model.BirthdayDate, 0, 2)
	seen := make(map[string]bool, 2)
	for _, in := range inputs {
		calendar := strings.TrimSpace(in.Calendar)
		if calendar == "" {
			calendar = "solar"
		}
		if calendar != "solar" && calendar != "lunar" {
			return nil, badRequest("生日类型无效")
		}
		if seen[calendar] {
			return nil, badRequest("公历和农历生日各只能填写一组")
		}
		seen[calendar] = true
		if in.Month == nil || in.Day == nil {
			return nil, badRequest("请填写生日月份和日期")
		}
		month, day := *in.Month, *in.Day
		if calendar == "solar" {
			if month < 1 || month > 12 {
				return nil, badRequest("月份需在 1–12 之间")
			}
			if day < 1 || day > solarMonthDays(month) {
				return nil, badRequest("生日日期无效")
			}
		} else {
			if month < 1 || month > 12 {
				return nil, badRequest("农历月份需在 1–12 之间")
			}
			if day < 1 || day > 30 {
				return nil, badRequest("农历日期需在 1–30 之间")
			}
		}
		d := model.BirthdayDate{
			Calendar: calendar,
			Month:    month,
			Day:      day,
		}
		if in.YearUnknown != nil {
			d.YearUnknown = *in.YearUnknown
		} else if in.BirthYear == nil {
			d.YearUnknown = true
		}
		if in.BirthYear != nil {
			if *in.BirthYear != 0 {
				if *in.BirthYear < 1900 || *in.BirthYear > time.Now().Year() {
					return nil, badRequest("出生年份无效")
				}
				d.BirthYear = in.BirthYear
				d.YearUnknown = false
			} else {
				d.BirthYear = nil
				d.YearUnknown = true
			}
		}
		if d.YearUnknown {
			d.BirthYear = nil
		}
		if calendar == "solar" {
			offset := 0
			if in.ReminderOffset != nil {
				if *in.ReminderOffset > 10080 {
					return nil, badRequest("提醒提前量无效")
				}
				if *in.ReminderOffset > 0 {
					offset = *in.ReminderOffset
				}
			}
			d.ReminderOffset = &offset
			d.CalendarEventID = existingEvents["solar"]
		} else if in.ReminderOffset != nil {
			if *in.ReminderOffset < 0 || *in.ReminderOffset > 10080 {
				return nil, badRequest("提醒提前量无效")
			}
			offset := *in.ReminderOffset
			d.ReminderOffset = &offset
		}
		out = append(out, d)
	}
	return out, nil
}

func solarMonthDays(month int) int {
	if month == 2 {
		return 29
	}
	return time.Date(2024, time.Month(month)+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

func hasSolarDate(dates []model.BirthdayDate) bool {
	for _, d := range dates {
		if d.Calendar == "solar" {
			return true
		}
	}
	return false
}

func parseBirthdayRange(fromRaw, toRaw string, loc *time.Location) (time.Time, time.Time, bool, *APIError) {
	fromRaw = strings.TrimSpace(fromRaw)
	toRaw = strings.TrimSpace(toRaw)
	if fromRaw == "" && toRaw == "" {
		return time.Time{}, time.Time{}, false, nil
	}
	if fromRaw == "" || toRaw == "" {
		return time.Time{}, time.Time{}, false, badRequest("from 和 to 需同时提供")
	}
	from, err := time.ParseInLocation("2006-01-02", fromRaw, loc)
	if err != nil {
		return time.Time{}, time.Time{}, false, badRequest("from 日期格式无效")
	}
	to, err := time.ParseInLocation("2006-01-02", toRaw, loc)
	if err != nil {
		return time.Time{}, time.Time{}, false, badRequest("to 日期格式无效")
	}
	if from.After(to) {
		return time.Time{}, time.Time{}, false, badRequest("from 不能晚于 to")
	}
	return from, to, true, nil
}

func decorateBirthday(b *model.Birthday, today time.Time) {
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())
	b.NextDate = nil
	b.DaysUntil = nil
	b.Age = nil
	b.NextCalendar = nil
	bestDays := 0
	for i := range b.Dates {
		d := &b.Dates[i]
		d.NextDate = nil
		d.DaysUntil = nil
		d.Age = nil
		next, lunarYear, ok := nextBirthdayOccurrence(*d, today)
		if !ok {
			continue
		}
		days := int(next.Sub(today).Hours() / 24)
		date := next.Format("2006-01-02")
		age := birthdayAge(*d, next.Year(), lunarYear)
		d.NextDate = &date
		d.DaysUntil = &days
		d.Age = age
		if b.NextDate == nil || days < bestDays {
			bestDays = days
			calendar := d.Calendar
			b.NextDate = &date
			b.DaysUntil = &bestDays
			b.Age = age
			b.NextCalendar = &calendar
		}
	}
}

func nextBirthdayOccurrence(d model.BirthdayDate, today time.Time) (time.Time, int, bool) {
	if d.Calendar == "lunar" {
		return holiday.NextLunarOccurrence(d.Month, d.Day, today)
	}
	next := nextBirthdayDate(d.Month, d.Day, today)
	return next, next.Year(), true
}

func birthdayAge(d model.BirthdayDate, solarYear, lunarYear int) *int {
	if d.YearUnknown || d.BirthYear == nil {
		return nil
	}
	year := solarYear
	if d.Calendar == "lunar" {
		year = lunarYear
	}
	age := year - *d.BirthYear
	if age < 0 {
		return nil
	}
	return &age
}

func birthdayOccurrencesInRange(b *model.Birthday, from, to time.Time) []model.BirthdayOccurrence {
	type agg struct {
		calendar string
		age      *int
		solar    bool
	}
	seen := make(map[string]*agg)
	add := func(date, calendar string, age *int) {
		cur := seen[date]
		if cur == nil {
			seen[date] = &agg{calendar: calendar, age: age, solar: calendar == "solar"}
			return
		}
		if cur.calendar == calendar {
			return
		}
		cur.calendar = "both"
		if calendar == "solar" {
			cur.solar = true
			cur.age = age
		}
	}
	loc := from.Location()
	for i := range b.Dates {
		d := b.Dates[i]
		if d.Calendar == "lunar" {
			for _, occ := range holiday.LunarOccurrences(d.Month, d.Day, from, to) {
				add(occ.Date.Format("2006-01-02"), "lunar", birthdayAge(d, occ.Date.Year(), occ.LunarYear))
			}
			continue
		}
		for year := from.Year(); year <= to.Year(); year++ {
			date := solarOccurrence(year, d.Month, d.Day, loc)
			if date.Before(from) || date.After(to) {
				continue
			}
			add(date.Format("2006-01-02"), "solar", birthdayAge(d, year, year))
		}
	}
	dates := make([]string, 0, len(seen))
	for date := range seen {
		dates = append(dates, date)
	}
	sort.Strings(dates)
	out := make([]model.BirthdayOccurrence, 0, len(dates))
	for _, date := range dates {
		item := seen[date]
		out = append(out, model.BirthdayOccurrence{Date: date, Calendar: item.calendar, Age: item.age})
	}
	return out
}

func solarOccurrence(year, month, day int, loc *time.Location) time.Time {
	last := time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, loc).Day()
	if day > last {
		day = last
	}
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, loc)
}

func nextBirthdayDate(month, day int, today time.Time) time.Time {
	for i := 0; i < 9; i++ {
		year := today.Year() + i
		candidate := solarOccurrence(year, month, day, today.Location())
		if !candidate.Before(today) {
			return candidate
		}
	}
	return today
}

func (s *Server) syncBirthdayEvent(ctx context.Context, b *model.Birthday) error {
	solarIndex := -1
	for i := range b.Dates {
		if b.Dates[i].Calendar == "solar" {
			solarIndex = i
			break
		}
	}
	if solarIndex < 0 {
		return nil
	}
	d := &b.Dates[solarIndex]
	title := b.Name + "的生日"
	anchor := birthdayAnchorDate(d.Month, d.Day)
	rule := `{"freq":"yearly","interval":1,"anchor":"due"}`
	var ev *model.CalendarEvent
	if d.CalendarEventID != nil {
		ev, _ = s.store.GetEvent(ctx, *d.CalendarEventID)
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
		d.CalendarEventID = &id
		return s.store.SetBirthdayDateEvent(ctx, d.ID, &id)
	}
	ev.Title = title
	ev.StartLocal = anchor + " 00:00:00"
	ev.AllDay = true
	ev.RecurrenceRule = &rule
	ev.Visibility = "household"
	return s.store.UpdateEvent(ctx, ev)
}

func (s *Server) syncBirthdayReminder(ctx context.Context, b *model.Birthday) error {
	loc, err := s.location(ctx)
	if err != nil {
		return err
	}
	now := time.Now().In(loc)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	type group struct {
		offset    int
		calendars map[string]bool
	}
	groups := make(map[string]*group, 2)
	for i := range b.Dates {
		d := b.Dates[i]
		if d.ReminderOffset == nil {
			continue
		}
		next, _, ok := nextBirthdayOccurrence(d, today)
		if !ok {
			continue
		}
		date := next.Format("2006-01-02")
		g := groups[date]
		if g == nil {
			g = &group{offset: *d.ReminderOffset, calendars: make(map[string]bool, 2)}
			groups[date] = g
		}
		if *d.ReminderOffset < g.offset {
			g.offset = *d.ReminderOffset
		}
		g.calendars[d.Calendar] = true
	}
	dates := make([]string, 0, len(groups))
	for date := range groups {
		dates = append(dates, date)
	}
	sort.Strings(dates)
	specs := make([]store.ReminderSpec, 0, len(dates))
	for _, date := range dates {
		g := groups[date]
		next, err := time.ParseInLocation("2006-01-02", date, loc)
		if err != nil {
			continue
		}
		at := time.Date(next.Year(), next.Month(), next.Day(), 9, 0, 0, 0, loc).Unix() - int64(g.offset)*60
		if at <= now.Unix() {
			continue
		}
		title := b.Name + "的生日"
		if g.calendars["solar"] && g.calendars["lunar"] {
			title += "（公历/农历同日）"
		} else if g.calendars["lunar"] {
			title += "（农历）"
		}
		occ := date
		specs = append(specs, store.ReminderSpec{
			UserID:         b.CreatedBy,
			RemindAt:       at,
			OccurrenceDate: &occ,
			Title:          title,
		})
	}
	return s.store.ReplaceSourceReminders(ctx, "birthday", b.ID, specs)
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
