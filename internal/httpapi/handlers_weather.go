package httpapi

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// weatherRequestTimeout 限制单次天气请求耗时，超时后由缓存兜底，避免接口一直卡住。
const weatherRequestTimeout = 6 * time.Second

// handleWeather 代理上游天气，位置优先级：查询坐标 → 个人偏好 → 家庭默认。
func (s *Server) handleWeather(w http.ResponseWriter, r *http.Request) {
	u, _ := currentUser(r)
	q := r.URL.Query()
	lat, lon, city, apiErr := s.weatherLocationFor(r.Context(), u.ID, q.Get("lat"), q.Get("lon"), q.Get("city"))
	if apiErr != nil {
		writeError(w, apiErr)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), weatherRequestTimeout)
	defer cancel()
	data, _, err := s.weather.Current(ctx, s.weatherProvider(r.Context()), lat, lon, city)
	if err != nil {
		writeError(w, &APIError{
			Status:  http.StatusServiceUnavailable,
			Code:    "weather.unavailable",
			Message: "天气服务暂时不可用，请稍后再试",
		})
		return
	}
	writeJSON(w, http.StatusOK, data)
}

// handleWeatherSources 返回可选的天气信息源列表。
func (s *Server) handleWeatherSources(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"items": s.weather.List()})
}

// weatherProvider 返回家庭设置的天气信息源；未设置时交给 Manager 使用默认源。
func (s *Server) weatherProvider(ctx context.Context) string {
	h, err := s.store.GetHousehold(ctx)
	if err != nil || h.DefaultWeatherProvider == nil {
		return ""
	}
	return *h.DefaultWeatherProvider
}

func (s *Server) weatherLocationFor(ctx context.Context, userID int64, latRaw, lonRaw, cityRaw string) (float64, float64, string, *APIError) {
	lat, latOK := parseCoord(latRaw, -90, 90)
	lon, lonOK := parseCoord(lonRaw, -180, 180)
	if latOK && lonOK {
		return lat, lon, strings.TrimSpace(cityRaw), nil
	}

	prefs, err := s.store.GetPreferences(ctx, userID)
	if err != nil {
		return 0, 0, "", serverError(err)
	}
	city := strings.TrimSpace(cityRaw)
	if city == "" && prefs.WeatherCity != nil {
		city = *prefs.WeatherCity
	}
	if prefs.WeatherLat != nil && prefs.WeatherLon != nil {
		return *prefs.WeatherLat, *prefs.WeatherLon, city, nil
	}

	h, err := s.store.GetHousehold(ctx)
	if err != nil {
		return 0, 0, "", serverError(err)
	}
	if city == "" && h.DefaultWeatherCity != nil {
		city = *h.DefaultWeatherCity
	}
	if h.DefaultWeatherLat != nil && h.DefaultWeatherLon != nil {
		return *h.DefaultWeatherLat, *h.DefaultWeatherLon, city, nil
	}
	return 0, 0, "", badRequest("请先在设置中指定天气城市")
}

func parseCoord(raw string, min, max float64) (float64, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, false
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil || v < min || v > max {
		return 0, false
	}
	return v, true
}
