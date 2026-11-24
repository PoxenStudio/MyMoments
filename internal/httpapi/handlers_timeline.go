package httpapi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"mymoments/internal/model"
	"mymoments/internal/recur"
	"mymoments/internal/store"
)

type timelineEntry struct {
	SourceType    string            `json:"sourceType"`
	SourceID      int64             `json:"sourceId"`
	OccurrenceKey string            `json:"occurrenceKey"`
	OccurredOn    string            `json:"occurredOn"`
	OccurredAt    *int64            `json:"occurredAt"`
	Title         string            `json:"title"`
	Summary       *string           `json:"summary"`
	MemberIDs     []int64           `json:"memberIds"`
	Members       []model.User      `json:"members"`
	Status        *string           `json:"status"`
	MediaPreview  []model.TaskMedia `json:"mediaPreview"`
	DetailTarget  map[string]any    `json:"detailTarget"`

	rank      int
	taskID    int64
	occurDate string
}

type timelineCursor struct {
	Key  string `json:"k"`
	On   string `json:"o"`
	At   int64  `json:"a"`
	Rank int    `json:"r"`
}

// handleTimeline 聚合待办、日历、生日、留言与便签，支持单向游标分页。
func (s *Server) handleTimeline(w http.ResponseWriter, r *http.Request) {
	u, _ := currentUser(r)
	q := r.URL.Query()
	loc, err := s.location(r.Context())
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	anchor := strings.TrimSpace(q.Get("anchor"))
	if anchor == "" {
		anchor = s.today()
	}
	if err := recur.ValidateDate(anchor); err != nil {
		writeError(w, badRequest("锚点日期格式需为 YYYY-MM-DD"))
		return
	}
	direction := "past"
	if q.Get("direction") == "future" {
		direction = "future"
	}
	limit := int(queryInt(q.Get("limit")))
	if limit <= 0 {
		limit = 30
	}
	if limit > 100 {
		limit = 100
	}
	types := parseTimelineTypes(q.Get("types"))
	memberID := queryInt(q.Get("member"))
	search := strings.TrimSpace(q.Get("q"))

	from, to := timelineWindow(anchor, direction)
	entries, partial, err := s.collectTimeline(r, u, direction, anchor, from, to, types, search, loc)
	if err != nil {
		writeError(w, serverError(err))
		return
	}

	// 按方向过滤边界日期。
	filtered := entries[:0]
	for _, e := range entries {
		if direction == "past" && e.OccurredOn > anchor {
			continue
		}
		if direction == "future" && e.OccurredOn < anchor {
			continue
		}
		if memberID > 0 && !containsID(e.MemberIDs, memberID) {
			continue
		}
		filtered = append(filtered, e)
	}
	entries = filtered

	if direction == "past" {
		sort.SliceStable(entries, func(i, j int) bool {
			return timelineLess(entries[i], entries[j], true)
		})
	} else {
		sort.SliceStable(entries, func(i, j int) bool {
			return timelineLess(entries[i], entries[j], false)
		})
	}

	cursor, apiErr := decodeTimelineCursor(q.Get("cursor"))
	if apiErr != nil {
		writeError(w, apiErr)
		return
	}
	if cursor != nil {
		entries = applyTimelineCursor(entries, cursor)
	}

	hasMore := len(entries) > limit
	page := entries
	if hasMore {
		page = entries[:limit]
	}
	if err := s.attachTimelineMedia(r.Context(), page); err != nil {
		writeError(w, serverError(err))
		return
	}
	next := ""
	if hasMore && len(page) > 0 {
		last := page[len(page)-1]
		next = encodeTimelineCursor(timelineCursor{
			Key: last.OccurrenceKey, On: last.OccurredOn, At: timelineAt(last), Rank: last.rank,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"items":      page,
		"anchor":     anchor,
		"direction":  direction,
		"nextCursor": next,
		"hasMore":    hasMore,
		"partial":    partial,
	})
}

func timelineWindow(anchor, direction string) (string, string) {
	if direction == "future" {
		return anchor, addYears(anchor, 2)
	}
	return addYears(anchor, -5), anchor
}

func parseTimelineTypes(raw string) map[string]bool {
	out := map[string]bool{}
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			out[part] = true
		}
	}
	if len(out) == 0 {
		for _, t := range []string{"task", "event", "birthday", "message", "note"} {
			out[t] = true
		}
	}
	return out
}

func (s *Server) collectTimeline(r *http.Request, u *model.User, direction, anchor, from, to string,
	types map[string]bool, search string, loc *time.Location) ([]timelineEntry, []string, error) {
	ctx := r.Context()
	entries := make([]timelineEntry, 0)
	partial := make([]string, 0)

	users, err := s.store.ListUsers(ctx)
	if err != nil {
		partial = append(partial, "member")
	}
	userMap := make(map[int64]model.User, len(users))
	for _, item := range users {
		userMap[item.ID] = item.Public()
	}

	if types["message"] {
		posts, err := s.store.ListTimelineMessages(ctx, u.ID)
		if err != nil {
			partial = append(partial, "message")
		} else {
			for _, p := range posts {
				entries = append(entries, messageEntry(p, false, loc, userMap))
			}
		}
		replies, err := s.store.ListTimelineReplies(ctx, u.ID)
		if err != nil {
			partial = append(partial, "message")
		} else {
			for _, p := range replies {
				entries = append(entries, messageEntry(p, true, loc, userMap))
			}
		}
	}

	if types["note"] {
		notes, err := s.store.ListTimelineNotes(ctx, u.ID)
		if err != nil {
			partial = append(partial, "note")
		} else {
			for _, n := range notes {
				entries = append(entries, noteEntry(n, loc, userMap))
			}
		}
	}

	if types["task"] {
		tasks, err := s.store.ListTasks(ctx, store.TaskFilter{
			ViewerID: u.ID, Scope: "open", From: from, To: to, Limit: 200,
		})
		if err != nil {
			partial = append(partial, "task")
		} else {
			for _, t := range tasks {
				if t.DueDate == nil {
					continue
				}
				entries = append(entries, taskEntry(t, s.today()))
			}
		}
		if direction == "past" {
			completions, err := s.store.ListTimelineCompletions(ctx, u.ID,
				startOfDayTS(loc, from), endOfDayTS(loc, anchor))
			if err != nil {
				partial = append(partial, "task")
			} else {
				for _, tc := range completions {
					assignees, err := s.store.TaskAssignees(ctx, tc.Task.ID)
					if err != nil {
						partial = append(partial, "task")
						continue
					}
					entries = append(entries, completionEntry(tc, assignees, loc))
				}
			}
		}
	}

	if types["event"] {
		events, err := s.store.ListEvents(ctx, u.ID, from, to, false)
		if err != nil {
			partial = append(partial, "event")
		} else {
			seriesIDs := make([]string, 0)
			for _, ev := range events {
				if ev.SeriesID != nil && ev.RecurrenceRule != nil {
					seriesIDs = append(seriesIDs, *ev.SeriesID)
				}
			}
			exceptions, err := s.store.LoadEventExceptions(ctx, seriesIDs)
			if err != nil {
				partial = append(partial, "event")
			} else {
				for _, base := range events {
					if base.Source == "birthday" {
						continue
					}
					for _, ev := range expandEvent(base, from, to, exceptions) {
						date := ev.StartLocal
						if len(date) >= 10 {
							date = date[:10]
						}
						if ev.OccurrenceDate != nil {
							date = *ev.OccurrenceDate
						}
						entries = append(entries, eventEntry(ev, date, loc))
					}
				}
			}
		}
	}

	if types["birthday"] {
		birthdays, err := s.store.ListBirthdays(ctx, store.BirthdayFilter{})
		if err != nil {
			partial = append(partial, "birthday")
		} else {
			for _, b := range birthdays {
				entries = append(entries, birthdayEntries(b, from, to, s.today(), userMap)...)
			}
		}
	}

	if search != "" {
		needle := strings.ToLower(search)
		kept := entries[:0]
		for _, e := range entries {
			hay := strings.ToLower(e.Title)
			if e.Summary != nil {
				hay += " " + strings.ToLower(*e.Summary)
			}
			if strings.Contains(hay, needle) {
				kept = append(kept, e)
			}
		}
		entries = kept
	}
	return entries, partial, nil
}

func messageEntry(p store.TimelinePost, reply bool, loc *time.Location, users map[int64]model.User) timelineEntry {
	occurredOn := time.Unix(p.CreatedAt, 0).In(loc).Format("2006-01-02")
	if p.OccurrenceDate != nil && *p.OccurrenceDate != "" {
		occurredOn = *p.OccurrenceDate
	}
	summary := snippet(p.Body, 90)
	title := p.AuthorName + " 在留言板留言"
	key := fmt.Sprintf("message:%d", p.ID)
	rank := 4
	sourceID := p.PostID
	if reply {
		title = p.AuthorName + " 回复了留言"
		key = fmt.Sprintf("reply:%d", p.ID)
		rank = 5
	}
	status := "record"
	at := p.CreatedAt
	memberIDs, members := entryMembers(p.AuthorID, p.AuthorName, users)
	return timelineEntry{
		SourceType: "message", SourceID: sourceID, OccurrenceKey: key,
		OccurredOn: occurredOn, OccurredAt: &at, Title: title, Summary: &summary,
		MemberIDs: memberIDs, Members: members,
		Status: &status, MediaPreview: []model.TaskMedia{},
		DetailTarget: map[string]any{"view": "messages", "open": sourceID, "date": occurredOn},
		rank:         rank,
	}
}

func noteEntry(n store.TimelineNote, loc *time.Location, users map[int64]model.User) timelineEntry {
	occurredOn := time.Unix(n.CreatedAt, 0).In(loc).Format("2006-01-02")
	if n.MemoryDate != nil && *n.MemoryDate != "" {
		occurredOn = *n.MemoryDate
	}
	title := n.AuthorName + " 写了一张便签"
	if n.Title != nil && strings.TrimSpace(*n.Title) != "" {
		title = *n.Title
	}
	body := ""
	if n.Body != nil {
		body = *n.Body
	}
	summary := snippet(body, 90)
	status := "record"
	at := n.CreatedAt
	memberIDs, members := entryMembers(n.AuthorID, n.AuthorName, users)
	return timelineEntry{
		SourceType: "note", SourceID: n.ID, OccurrenceKey: fmt.Sprintf("note:%d", n.ID),
		OccurredOn: occurredOn, OccurredAt: &at, Title: title, Summary: &summary,
		MemberIDs: memberIDs, Members: members,
		Status: &status, MediaPreview: []model.TaskMedia{},
		DetailTarget: map[string]any{"view": "notes", "open": n.ID, "date": occurredOn},
		rank:         6,
	}
}

func entryMembers(id int64, name string, users map[int64]model.User) ([]int64, []model.User) {
	if u, ok := users[id]; ok {
		return []int64{id}, []model.User{u}
	}
	return []int64{id}, []model.User{{ID: id, DisplayName: name}}
}

func taskEntry(t model.Task, today string) timelineEntry {
	if t.DueDate == nil {
		return timelineEntry{}
	}
	status := "planned"
	if *t.DueDate < today {
		status = "overdue"
	}
	summary := ""
	if t.Description != nil {
		summary = snippet(*t.Description, 90)
	}
	memberIDs := make([]int64, 0, len(t.Assignees))
	for _, a := range t.Assignees {
		memberIDs = append(memberIDs, a.ID)
	}
	return timelineEntry{
		SourceType: "task", SourceID: t.ID, OccurrenceKey: fmt.Sprintf("task:%d:%s", t.ID, *t.DueDate),
		OccurredOn: *t.DueDate, Title: t.Title, Summary: ptrString(summary),
		MemberIDs: memberIDs, Members: t.Assignees, Status: &status,
		MediaPreview: []model.TaskMedia{},
		DetailTarget: map[string]any{"view": "tasks", "open": t.ID, "date": *t.DueDate},
		rank:         1, taskID: t.ID, occurDate: *t.DueDate,
	}
}

func completionEntry(tc store.TimelineCompletion, assignees []model.User, loc *time.Location) timelineEntry {
	date := time.Unix(tc.CompletedAt, 0).In(loc).Format("2006-01-02")
	summary := "已完成"
	if tc.DoneByName != "" {
		summary = "由 " + tc.DoneByName + " 完成"
	}
	memberIDs := make([]int64, 0, len(assignees))
	for _, a := range assignees {
		memberIDs = append(memberIDs, a.ID)
	}
	status := "completed"
	at := tc.CompletedAt
	tc.Task.Assignees = assignees
	return timelineEntry{
		SourceType: "task_completion", SourceID: tc.Task.ID,
		OccurrenceKey: fmt.Sprintf("task_completion:%d", tc.CompletionID),
		OccurredOn:    date, OccurredAt: &at, Title: tc.Task.Title, Summary: &summary,
		MemberIDs: memberIDs, Members: assignees, Status: &status,
		MediaPreview: []model.TaskMedia{},
		DetailTarget: map[string]any{"view": "tasks", "open": tc.Task.ID, "date": date},
		rank:         0, taskID: tc.Task.ID, occurDate: tc.OccurrenceDate,
	}
}

func eventEntry(ev model.CalendarEvent, date string, loc *time.Location) timelineEntry {
	summary := ""
	if ev.Location != nil {
		summary = *ev.Location
	} else if ev.Description != nil {
		summary = snippet(*ev.Description, 90)
	}
	memberIDs := make([]int64, 0, len(ev.Assignees))
	for _, a := range ev.Assignees {
		memberIDs = append(memberIDs, a.ID)
	}
	status := "past"
	if date >= todayIn(loc) {
		status = "planned"
	}
	var at *int64
	if len(ev.StartLocal) >= 19 {
		if ts, err := time.ParseInLocation("2006-01-02 15:04:05", ev.StartLocal, loc); err == nil {
			v := ts.Unix()
			at = &v
		}
	}
	return timelineEntry{
		SourceType: "event", SourceID: ev.ID, OccurrenceKey: fmt.Sprintf("event:%d:%s", ev.ID, date),
		OccurredOn: date, OccurredAt: at, Title: ev.Title, Summary: ptrString(summary),
		MemberIDs: memberIDs, Members: ev.Assignees, Status: &status,
		MediaPreview: []model.TaskMedia{},
		DetailTarget: map[string]any{"view": "calendar", "open": ev.ID, "date": date},
		rank:         2,
	}
}

func birthdayEntries(b model.Birthday, from, to, today string, users map[int64]model.User) []timelineEntry {
	fromYear := yearOf(from)
	toYear := yearOf(to)
	out := make([]timelineEntry, 0, toYear-fromYear+1)
	members := []model.User{}
	memberIDs := []int64{}
	if b.UserID != nil {
		if u, ok := users[*b.UserID]; ok {
			members = append(members, u)
			memberIDs = append(memberIDs, u.ID)
		}
	}
	for year := fromYear; year <= toYear; year++ {
		date := birthdayDate(year, b.Month, b.Day)
		if date < from || date > to {
			continue
		}
		summary := ""
		if b.BirthYear != nil {
			age := year - *b.BirthYear
			if age > 0 {
				summary = fmt.Sprintf("%d 岁生日", age)
			}
		}
		status := "past"
		if date >= today {
			status = "upcoming"
		}
		out = append(out, timelineEntry{
			SourceType: "birthday", SourceID: b.ID, OccurrenceKey: fmt.Sprintf("birthday:%d:%d", b.ID, year),
			OccurredOn: date, Title: b.Name + " 的生日", Summary: ptrString(summary),
			MemberIDs: memberIDs, Members: members, Status: &status,
			MediaPreview: []model.TaskMedia{},
			DetailTarget: map[string]any{"view": "family", "open": b.ID, "date": date},
			rank:         3,
		})
	}
	return out
}

func (s *Server) attachTimelineMedia(ctx context.Context, entries []timelineEntry) error {
	for i := range entries {
		if entries[i].taskID == 0 {
			continue
		}
		occurrence := entries[i].occurDate
		var occPtr *string
		if occurrence != "" {
			occPtr = &occurrence
		}
		media, err := s.store.ListMedia(ctx, entries[i].taskID, occPtr)
		if err != nil {
			return err
		}
		preview := make([]model.TaskMedia, 0, 3)
		for _, m := range media {
			if m.MediaType != "image" {
				continue
			}
			preview = append(preview, m)
			if len(preview) == 3 {
				break
			}
		}
		entries[i].MediaPreview = preview
	}
	return nil
}

func timelineAt(e timelineEntry) int64 {
	if e.OccurredAt != nil {
		return *e.OccurredAt
	}
	return 0
}

func timelineLess(a, b timelineEntry, desc bool) bool {
	if a.OccurredOn != b.OccurredOn {
		if desc {
			return a.OccurredOn > b.OccurredOn
		}
		return a.OccurredOn < b.OccurredOn
	}
	if timelineAt(a) != timelineAt(b) {
		if desc {
			return timelineAt(a) > timelineAt(b)
		}
		return timelineAt(a) < timelineAt(b)
	}
	if a.rank != b.rank {
		if desc {
			return a.rank > b.rank
		}
		return a.rank < b.rank
	}
	if desc {
		return a.OccurrenceKey > b.OccurrenceKey
	}
	return a.OccurrenceKey < b.OccurrenceKey
}

func applyTimelineCursor(entries []timelineEntry, c *timelineCursor) []timelineEntry {
	for i := range entries {
		if entries[i].OccurrenceKey == c.Key {
			return entries[i+1:]
		}
	}
	// 游标对应条目已不可见时，退回按排序键过滤。
	kept := entries[:0]
	for _, e := range entries {
		if afterCursor(e, c) {
			kept = append(kept, e)
		}
	}
	return kept
}

func afterCursor(e timelineEntry, c *timelineCursor) bool {
	if e.OccurredOn != c.On {
		return e.OccurredOn < c.On
	}
	if timelineAt(e) != c.At {
		return timelineAt(e) < c.At
	}
	if e.rank != c.Rank {
		return e.rank < c.Rank
	}
	return e.OccurrenceKey < c.Key
}

func encodeTimelineCursor(c timelineCursor) string {
	b, _ := json.Marshal(c)
	return base64.RawURLEncoding.EncodeToString(b)
}

func decodeTimelineCursor(raw string) (*timelineCursor, *APIError) {
	if raw == "" {
		return nil, nil
	}
	b, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil, badRequest("游标无效")
	}
	var c timelineCursor
	if err := json.Unmarshal(b, &c); err != nil || c.Key == "" || c.On == "" {
		return nil, badRequest("游标无效")
	}
	return &c, nil
}

func snippet(text string, limit int) string {
	text = strings.TrimSpace(strings.ReplaceAll(text, "\r", ""))
	if utf8.RuneCountInString(text) <= limit {
		return text
	}
	runes := []rune(text)
	return string(runes[:limit]) + "…"
}

func ptrString(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}

func addYears(date string, years int) string {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return date
	}
	return t.AddDate(years, 0, 0).Format("2006-01-02")
}

func yearOf(date string) int {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return 0
	}
	return t.Year()
}

func startOfDayTS(loc *time.Location, date string) int64 {
	t, err := time.ParseInLocation("2006-01-02", date, loc)
	if err != nil {
		return 0
	}
	return t.Unix()
}

func endOfDayTS(loc *time.Location, date string) int64 {
	return startOfDayTS(loc, date) + 86399
}

func todayIn(loc *time.Location) string {
	return time.Now().In(loc).Format("2006-01-02")
}

func birthdayDate(year, month, day int) string {
	if month == 2 && day == 29 && !isLeapYear(year) {
		day = 28
	}
	return fmt.Sprintf("%04d-%02d-%02d", year, month, day)
}

func isLeapYear(year int) bool {
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}
