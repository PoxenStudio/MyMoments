package httpapi

import (
	"errors"
	"net/http"

	"mymoments/internal/store"
)

func (s *Server) handleNotificationsList(w http.ResponseWriter, r *http.Request) {
	u, _ := currentUser(r)
	items, err := s.store.ListNotifications(r.Context(), u.ID)
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	unread := 0
	for _, it := range items {
		if it.ReadAt == nil {
			unread++
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "unread": unread})
}

func (s *Server) handleNotificationRead(w http.ResponseWriter, r *http.Request) {
	u, _ := currentUser(r)
	id, err := parseID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	if err := s.store.MarkNotificationRead(r.Context(), u.ID, id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, notFound("通知不存在"))
			return
		}
		writeError(w, serverError(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleNotificationsReadAll(w http.ResponseWriter, r *http.Request) {
	u, _ := currentUser(r)
	n, err := s.store.MarkAllNotificationsRead(r.Context(), u.ID)
	if err != nil {
		writeError(w, serverError(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"updated": n})
}
