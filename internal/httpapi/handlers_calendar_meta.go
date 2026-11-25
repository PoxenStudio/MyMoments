package httpapi

import (
	"net/http"
	"strings"
	"time"
)

// handleCalendarMeta 返回指定日期范围内的农历、节日与休假信息。
func (s *Server) handleCalendarMeta(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	fromRaw := strings.TrimSpace(q.Get("from"))
	toRaw := strings.TrimSpace(q.Get("to"))
	if fromRaw == "" || toRaw == "" {
		writeError(w, badRequest("请提供 from 与 to 日期参数"))
		return
	}
	const layout = "2006-01-02"
	from, err := time.Parse(layout, fromRaw)
	if err != nil {
		writeError(w, badRequest("日期格式应为 YYYY-MM-DD"))
		return
	}
	to, err := time.Parse(layout, toRaw)
	if err != nil {
		writeError(w, badRequest("日期格式应为 YYYY-MM-DD"))
		return
	}
	days, err := s.holidays.Range(r.Context(), from, to)
	if err != nil {
		writeError(w, badRequest(err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"days": days})
}
