package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"mymoments/internal/model"
	"mymoments/internal/store"
)

type noteItemInput struct {
	Text    string `json:"text"`
	Checked bool   `json:"checked"`
}

type noteInput struct {
	Title       *string          `json:"title"`
	Body        *string          `json:"body"`
	Color       *string          `json:"color"`
	Category    *string          `json:"category"`
	MemoryDate  *string          `json:"memoryDate"`
	Visibility  *string          `json:"visibility"`
	AudienceIDs *[]int64         `json:"audienceIds"`
	Mentions    *[]mentionInput  `json:"mentions"`
	Pinned      *bool            `json:"pinned"`
	Items       *[]noteItemInput `json:"items"`
}

func (s *Server) handleNotesList(w http.ResponseWriter, r *http.Request) {
	u, _ := currentUser(r)
	q := r.URL.Query()
	items, err := s.store.ListNotes(r.Context(), store.NoteFilter{
		ViewerID:   u.ID,
		Query:      q.Get("q"),
		Category:   q.Get("category"),
		PinnedOnly: q.Get("pinned") == "1",
		Limit:      int(queryInt(q.Get("limit"))),
		Offset:     int(queryInt(q.Get("offset"))),
	})
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleNoteGet(w http.ResponseWriter, r *http.Request) {
	u, _ := currentUser(r)
	n, ok := s.loadNoteFor(r, w)
	if !ok {
		return
	}
	if !canReadNote(u, n) {
		writeError(w, notFound("便签不存在"))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"note": n})
}

func (s *Server) handleNoteCreate(w http.ResponseWriter, r *http.Request) {
	u, _ := currentUser(r)
	var in noteInput
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, err)
		return
	}
	n := &model.Note{AuthorID: u.ID, Visibility: "household"}
	if apiErr := s.applyNoteInput(r, n, &in, true); apiErr != nil {
		writeError(w, apiErr)
		return
	}
	mentions, apiErr := s.resolveMentions(r.Context(), u.ID, noteMentionSource(n), valueMentions(in.Mentions))
	if apiErr != nil {
		writeError(w, apiErr)
		return
	}
	id, err := s.store.CreateNote(r.Context(), n, mentions)
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	created, err := s.store.GetNote(r.Context(), id)
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"note": created})
}

func (s *Server) handleNotePatch(w http.ResponseWriter, r *http.Request) {
	u, _ := currentUser(r)
	n, ok := s.loadNoteFor(r, w)
	if !ok {
		return
	}
	if !canReadNote(u, n) {
		writeError(w, notFound("便签不存在"))
		return
	}
	if !canWriteNote(u, n) {
		writeError(w, forbidden("没有修改该便签的权限"))
		return
	}
	var in noteInput
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, err)
		return
	}
	if (in.Visibility != nil || in.AudienceIDs != nil) && n.AuthorID != u.ID && u.Role != "admin" {
		writeError(w, forbidden("只有创建者或管理员可以修改可见范围"))
		return
	}
	if apiErr := s.applyNoteInput(r, n, &in, false); apiErr != nil {
		writeError(w, apiErr)
		return
	}
	mentions, apiErr := s.resolveMentions(r.Context(), u.ID, noteMentionSource(n), valueMentions(in.Mentions))
	if apiErr != nil {
		writeError(w, apiErr)
		return
	}
	if err := s.store.UpdateNote(r.Context(), n, mentions); err != nil {
		writeError(w, serverError(err))
		return
	}
	updated, err := s.store.GetNote(r.Context(), n.ID)
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"note": updated})
}

func (s *Server) handleNoteDelete(w http.ResponseWriter, r *http.Request) {
	u, _ := currentUser(r)
	n, ok := s.loadNoteFor(r, w)
	if !ok {
		return
	}
	if !canReadNote(u, n) {
		writeError(w, notFound("便签不存在"))
		return
	}
	if !canDeleteNote(u, n) {
		writeError(w, forbidden("只有创建者或管理员可以删除该便签"))
		return
	}
	if err := s.store.DeleteNote(r.Context(), n.ID); err != nil {
		writeError(w, serverError(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) loadNoteFor(r *http.Request, w http.ResponseWriter) (*model.Note, bool) {
	id, err := parseID(r, "id")
	if err != nil {
		writeError(w, err)
		return nil, false
	}
	n, err := s.store.GetNote(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, notFound("便签不存在"))
			return nil, false
		}
		writeError(w, serverError(err))
		return nil, false
	}
	return n, true
}

func (s *Server) applyNoteInput(r *http.Request, n *model.Note, in *noteInput, create bool) *APIError {
	if in.Title != nil {
		title := strings.TrimSpace(*in.Title)
		if len([]rune(title)) > 120 {
			return badRequest("便签标题不能超过 120 个字符")
		}
		n.Title = nonEmptyPtr(title)
	}
	if in.Body != nil {
		body := strings.TrimSpace(*in.Body)
		if len([]rune(body)) > 10000 {
			return badRequest("便签正文不能超过 10000 个字符")
		}
		n.Body = nonEmptyPtr(body)
	}
	if in.Color != nil {
		color := strings.TrimSpace(*in.Color)
		if len(color) > 32 {
			return badRequest("便签颜色无效")
		}
		n.Color = nonEmptyPtr(color)
	}
	if in.Category != nil {
		category := strings.TrimSpace(*in.Category)
		if len([]rune(category)) > 40 {
			return badRequest("便签分类不能超过 40 个字符")
		}
		n.Category = nonEmptyPtr(category)
	}
	if in.MemoryDate != nil {
		date := strings.TrimSpace(*in.MemoryDate)
		if apiErr := s.validatePastDate(date, "回忆日期"); apiErr != nil {
			return apiErr
		}
		n.MemoryDate = nonEmptyPtr(date)
	}
	if in.Pinned != nil {
		n.Pinned = *in.Pinned
	}
	if in.Visibility != nil {
		if apiErr := validateVisibility(*in.Visibility); apiErr != nil {
			return apiErr
		}
		n.Visibility = *in.Visibility
	}
	if in.AudienceIDs != nil {
		ids, apiErr := s.resolveAudienceIDs(r.Context(), *in.AudienceIDs)
		if apiErr != nil {
			return apiErr
		}
		n.AudienceIDs = ids
	}
	if in.Items != nil {
		if len(*in.Items) > 100 {
			return badRequest("检查清单项过多")
		}
		items := make([]model.NoteItem, 0, len(*in.Items))
		for _, it := range *in.Items {
			text := strings.TrimSpace(it.Text)
			if text == "" {
				continue
			}
			if len([]rune(text)) > 200 {
				return badRequest("检查清单单项不能超过 200 个字符")
			}
			items = append(items, model.NoteItem{Text: text, Checked: it.Checked})
		}
		n.Items = items
	}
	if n.Visibility != "selected" {
		n.AudienceIDs = []int64{}
	} else if len(n.AudienceIDs) == 0 {
		return badRequest("指定成员可见需至少选择一位成员")
	}
	if create && n.Title == nil && n.Body == nil && len(n.Items) == 0 {
		return badRequest("请填写便签标题、正文或检查清单")
	}
	return nil
}

func noteMentionSource(n *model.Note) string {
	parts := make([]string, 0, 2)
	if n.Title != nil {
		parts = append(parts, *n.Title)
	}
	if n.Body != nil {
		parts = append(parts, *n.Body)
	}
	return strings.Join(parts, "\n")
}
