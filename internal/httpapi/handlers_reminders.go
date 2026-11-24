package httpapi

import (
	"errors"
	"net/http"

	"mymoments/internal/store"
)

func (s *Server) handleRemindersList(w http.ResponseWriter, r *http.Request) {
	u, _ := currentUser(r)
	status := r.URL.Query().Get("status")
	if status == "" {
		status = "pending"
	}
	items, err := s.store.ListReminders(r.Context(), u.ID, status)
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleReminderPatch(w http.ResponseWriter, r *http.Request) {
	u, _ := currentUser(r)
	id, err := parseID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	var body struct {
		Status string `json:"status"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, err)
		return
	}
	if body.Status != "dismissed" && body.Status != "delivered" {
		writeError(w, badRequest("提醒状态无效"))
		return
	}
	if err := s.store.UpdateReminderStatus(r.Context(), u.ID, id, body.Status); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, notFound("提醒不存在"))
			return
		}
		writeError(w, serverError(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
