package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"mymoments/internal/model"
	"mymoments/internal/store"
)

type messageInput struct {
	Body           *string  `json:"body"`
	OccurrenceDate *string  `json:"occurrenceDate"`
	Visibility     *string  `json:"visibility"`
	AudienceIDs    *[]int64 `json:"audienceIds"`
	MentionIDs     *[]int64 `json:"mentionIds"`
	Pinned         *bool    `json:"pinned"`
}

type replyInput struct {
	Body       *string  `json:"body"`
	MentionIDs *[]int64 `json:"mentionIds"`
}

func (s *Server) handleMessagesList(w http.ResponseWriter, r *http.Request) {
	u, _ := currentUser(r)
	q := r.URL.Query()
	filter := q.Get("filter")
	items, err := s.store.ListMessages(r.Context(), store.MessageFilter{
		ViewerID:     u.ID,
		Query:        q.Get("q"),
		MentionsOnly: filter == "mentions",
		MineOnly:     filter == "mine",
		PinnedOnly:   q.Get("pinned") == "1",
		Limit:        int(queryInt(q.Get("limit"))),
		Offset:       int(queryInt(q.Get("offset"))),
	})
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleMessageGet(w http.ResponseWriter, r *http.Request) {
	u, _ := currentUser(r)
	m, ok := s.loadMessageFor(r, w)
	if !ok {
		return
	}
	if !canReadMessage(u, m) {
		writeError(w, notFound("留言不存在"))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"message": m})
}

func (s *Server) handleMessageCreate(w http.ResponseWriter, r *http.Request) {
	u, _ := currentUser(r)
	var in messageInput
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, err)
		return
	}
	m := &model.MessagePost{AuthorID: u.ID, Visibility: "household"}
	if apiErr := s.applyMessageInput(r, m, &in, true); apiErr != nil {
		writeError(w, apiErr)
		return
	}
	mentionIDs, apiErr := s.resolveMentionIDs(r.Context(), m.Body, valueIDs(in.MentionIDs))
	if apiErr != nil {
		writeError(w, apiErr)
		return
	}
	id, err := s.store.CreateMessage(r.Context(), m, mentionIDs)
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	created, err := s.store.GetMessage(r.Context(), id)
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"message": created})
}

func (s *Server) handleMessagePatch(w http.ResponseWriter, r *http.Request) {
	u, _ := currentUser(r)
	m, ok := s.loadMessageFor(r, w)
	if !ok {
		return
	}
	if !canReadMessage(u, m) {
		writeError(w, notFound("留言不存在"))
		return
	}
	if !canWriteMessage(u, m) {
		writeError(w, forbidden("没有修改该留言的权限"))
		return
	}
	var in messageInput
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, err)
		return
	}
	if (in.Visibility != nil || in.AudienceIDs != nil) && m.AuthorID != u.ID && u.Role != "admin" {
		writeError(w, forbidden("只有创建者或管理员可以修改可见范围"))
		return
	}
	if apiErr := s.applyMessageInput(r, m, &in, false); apiErr != nil {
		writeError(w, apiErr)
		return
	}
	mentionIDs, apiErr := s.resolveMentionIDs(r.Context(), m.Body, valueIDs(in.MentionIDs))
	if apiErr != nil {
		writeError(w, apiErr)
		return
	}
	if m.Visibility != "selected" {
		m.AudienceIDs = []int64{}
	}
	if err := s.store.UpdateMessage(r.Context(), m, mentionIDs); err != nil {
		writeError(w, serverError(err))
		return
	}
	updated, err := s.store.GetMessage(r.Context(), m.ID)
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"message": updated})
}

func (s *Server) handleMessageDelete(w http.ResponseWriter, r *http.Request) {
	u, _ := currentUser(r)
	m, ok := s.loadMessageFor(r, w)
	if !ok {
		return
	}
	if !canReadMessage(u, m) {
		writeError(w, notFound("留言不存在"))
		return
	}
	if !canDeleteMessage(u, m) {
		writeError(w, forbidden("只有创建者或管理员可以删除该留言"))
		return
	}
	if err := s.store.DeleteMessage(r.Context(), m.ID); err != nil {
		writeError(w, serverError(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleMessageReply(w http.ResponseWriter, r *http.Request) {
	u, _ := currentUser(r)
	m, ok := s.loadMessageFor(r, w)
	if !ok {
		return
	}
	if !canReadMessage(u, m) {
		writeError(w, notFound("留言不存在"))
		return
	}
	var in replyInput
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, err)
		return
	}
	body := ""
	if in.Body != nil {
		body = strings.TrimSpace(*in.Body)
	}
	if body == "" || len([]rune(body)) > 2000 {
		writeError(w, badRequest("回复内容需为 1–2000 个字符"))
		return
	}
	mentionIDs, apiErr := s.resolveMentionIDs(r.Context(), body, valueIDs(in.MentionIDs))
	if apiErr != nil {
		writeError(w, apiErr)
		return
	}
	reply := &model.MessageReply{PostID: m.ID, AuthorID: u.ID, Body: body}
	if err := s.store.CreateReply(r.Context(), reply, mentionIDs, m.Visibility, m.AudienceIDs); err != nil {
		writeError(w, serverError(err))
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"reply": reply})
}

func (s *Server) loadMessageFor(r *http.Request, w http.ResponseWriter) (*model.MessagePost, bool) {
	id, err := parseID(r, "id")
	if err != nil {
		writeError(w, err)
		return nil, false
	}
	m, err := s.store.GetMessage(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, notFound("留言不存在"))
			return nil, false
		}
		writeError(w, serverError(err))
		return nil, false
	}
	return m, true
}

func (s *Server) applyMessageInput(r *http.Request, m *model.MessagePost, in *messageInput, create bool) *APIError {
	if in.Body != nil {
		body := strings.TrimSpace(*in.Body)
		if body == "" || len([]rune(body)) > 5000 {
			return badRequest("留言正文需为 1–5000 个字符")
		}
		m.Body = body
	} else if create {
		return badRequest("请填写留言内容")
	}
	if in.OccurrenceDate != nil {
		date := strings.TrimSpace(*in.OccurrenceDate)
		if apiErr := s.validatePastDate(date, "发生日期"); apiErr != nil {
			return apiErr
		}
		m.OccurrenceDate = nonEmptyPtr(date)
	}
	if in.Pinned != nil {
		m.Pinned = *in.Pinned
	}
	if in.Visibility != nil {
		if apiErr := validateVisibility(*in.Visibility); apiErr != nil {
			return apiErr
		}
		m.Visibility = *in.Visibility
	}
	if in.AudienceIDs != nil {
		ids, apiErr := s.resolveAudienceIDs(r.Context(), *in.AudienceIDs)
		if apiErr != nil {
			return apiErr
		}
		m.AudienceIDs = ids
	}
	if m.Visibility != "selected" {
		m.AudienceIDs = []int64{}
	} else if len(m.AudienceIDs) == 0 {
		return badRequest("指定成员可见需至少选择一位成员")
	}
	return nil
}

func valueIDs(ids *[]int64) []int64 {
	if ids == nil {
		return nil
	}
	return *ids
}
