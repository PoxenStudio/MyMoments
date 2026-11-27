package httpapi

import (
	"context"
	"net/http"
	"time"
)

// geoLookupTimeout 限制出口 IP 反查的总耗时，超时后让前端改用手动选择。
const geoLookupTimeout = 6 * time.Second

// handleGeoIP 通过服务端出口 IP 反查所在城市。
// 浏览器定位在 HTTP 页面不可用或浏览器直连上游被 CORS / CSP 拦截时，由前端回退到该接口。
func (s *Server) handleGeoIP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), geoLookupTimeout)
	defer cancel()
	res, err := s.geo.Locate(ctx)
	if err != nil {
		writeError(w, &APIError{
			Status:  http.StatusServiceUnavailable,
			Code:    "geo.unavailable",
			Message: "无法通过出口 IP 识别当前位置，请手动选择城市",
		})
		return
	}
	writeJSON(w, http.StatusOK, res)
}
