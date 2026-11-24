package httpapi

import (
	"net/http"
	"strings"

	"mymoments/internal/store"
)

type searchResult struct {
	Type    string  `json:"type"`
	ID      int64   `json:"id"`
	Title   string  `json:"title"`
	Summary *string `json:"summary"`
	Date    *string `json:"date"`
}

// handleSearch 跨模块搜索当前用户可见的内容。
func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	u, _ := currentUser(r)
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		writeJSON(w, http.StatusOK, map[string]any{"items": []searchResult{}})
		return
	}
	if len([]rune(q)) > 50 {
		writeError(w, badRequest("搜索词过长"))
		return
	}
	items := make([]searchResult, 0, 25)

	tasks, err := s.store.ListTasks(r.Context(), store.TaskFilter{ViewerID: u.ID, Scope: "all", Query: q, Limit: 5})
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	for _, t := range tasks {
		items = append(items, searchResult{Type: "task", ID: t.ID, Title: t.Title, Summary: t.Description, Date: t.DueDate})
	}

	today := s.today()
	events, err := s.store.ListEvents(r.Context(), u.ID, datePlusDays(today, -365), datePlusDays(today, 365), false)
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	lower := strings.ToLower(q)
	matched := 0
	for _, ev := range events {
		if matched >= 5 {
			break
		}
		haystack := strings.ToLower(ev.Title + " " + strPtr(ev.Description) + " " + strPtr(ev.Location))
		if !strings.Contains(haystack, lower) {
			continue
		}
		date := ev.StartLocal
		if len(date) > 10 {
			date = date[:10]
		}
		summary := ev.Location
		if summary == nil {
			summary = ev.Description
		}
		items = append(items, searchResult{Type: "event", ID: ev.ID, Title: ev.Title, Summary: summary, Date: &date})
		matched++
	}

	contacts, err := s.store.ListContacts(r.Context(), store.ContactFilter{ViewerID: u.ID, Query: q})
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	for i, c := range contacts {
		if i >= 5 {
			break
		}
		items = append(items, searchResult{Type: "contact", ID: c.ID, Title: c.Name, Summary: c.Organization})
	}

	messages, err := s.store.ListMessages(r.Context(), store.MessageFilter{ViewerID: u.ID, Query: q, Limit: 5})
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	for _, m := range messages {
		title := truncate(strings.TrimSpace(m.Body), 80)
		summary := m.AuthorName
		date := m.OccurrenceDate
		items = append(items, searchResult{Type: "message", ID: m.ID, Title: title, Summary: &summary, Date: date})
	}

	notes, err := s.store.ListNotes(r.Context(), store.NoteFilter{ViewerID: u.ID, Query: q, Limit: 5})
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	for i := range notes {
		note := notes[i]
		title := "无标题便签"
		if note.Title != nil && strings.TrimSpace(*note.Title) != "" {
			title = *note.Title
		}
		items = append(items, searchResult{Type: "note", ID: note.ID, Title: title, Summary: note.Body})
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func strPtr(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
