package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"mymoments/internal/config"
)

func TestDashboardAggregationAndLayout(t *testing.T) {
	base, admin, _ := setupHousehold(t)
	today := todayString()

	if status, body := call(t, admin, http.MethodPost, base+"/tasks", map[string]any{
		"title": "买菜", "dueDate": today,
	}); status != http.StatusCreated {
		t.Fatalf("create task: %d %v", status, body)
	}
	if status, body := call(t, admin, http.MethodPost, base+"/events", map[string]any{
		"title": "晚饭", "startLocal": today + " 18:00:00",
	}); status != http.StatusCreated {
		t.Fatalf("create event: %d %v", status, body)
	}
	if status, body := call(t, admin, http.MethodPost, base+"/messages", map[string]any{
		"body": "今天天气很好",
	}); status != http.StatusCreated {
		t.Fatalf("create message: %d %v", status, body)
	}
	if status, body := call(t, admin, http.MethodPost, base+"/notes", map[string]any{
		"title": "周末计划",
	}); status != http.StatusCreated {
		t.Fatalf("create note: %d %v", status, body)
	}
	if status, body := call(t, admin, http.MethodPost, base+"/birthdays", map[string]any{
		"name": "妈妈", "month": 12, "day": 1, "yearUnknown": true,
	}); status != http.StatusCreated {
		t.Fatalf("create birthday: %d %v", status, body)
	}

	status, body := call(t, admin, http.MethodGet, base+"/dashboard", nil)
	if status != http.StatusOK {
		t.Fatalf("dashboard: %d %v", status, body)
	}
	if body["layoutSource"] != "default" || body["today"] != today {
		t.Fatalf("unexpected dashboard meta: %v", body)
	}
	layout := body["layout"].([]any)
	if len(layout) != len(dashboardCardDefs) {
		t.Fatalf("unexpected layout size: %d", len(layout))
	}
	first := layout[0].(map[string]any)
	if first["id"] != "overview" || first["visible"] != true {
		t.Fatalf("overview must be pinned visible: %v", first)
	}
	cards := map[string]map[string]any{}
	for _, raw := range body["cards"].([]any) {
		card := raw.(map[string]any)
		cards[card["id"].(string)] = card
	}
	overview := cards["overview"]
	if overview["data"] == nil {
		t.Fatalf("overview data missing: %v", overview)
	}
	counts := overview["data"].(map[string]any)["counts"].(map[string]any)
	if counts["open"].(float64) != 1 || counts["today"].(float64) != 1 {
		t.Fatalf("unexpected counts: %v", counts)
	}
	if cards["timeline"]["visible"] != false || cards["timeline"]["data"] != nil {
		t.Fatalf("timeline should be hidden by default: %v", cards["timeline"])
	}
	if cards["tasks"]["data"] == nil || cards["calendar"]["data"] == nil || cards["birthdays"]["data"] == nil {
		t.Fatalf("default cards missing data: %v", cards)
	}

	// 调整布局：tasks 放大、天气开启、时光轴置顶到待办之前。
	newLayout := []map[string]any{
		{"id": "overview", "visible": true, "size": "wide", "background": "default"},
		{"id": "timeline", "visible": true, "size": "wide", "background": "sage"},
		{"id": "tasks", "visible": true, "size": "wide", "background": "warm"},
	}
	status, body = call(t, admin, http.MethodPut, base+"/dashboard/layout", map[string]any{"layout": newLayout})
	if status != http.StatusOK || body["layoutSource"] != "personal" {
		t.Fatalf("save layout: %d %v", status, body)
	}
	saved := body["layout"].([]any)
	if saved[0].(map[string]any)["id"] != "overview" || saved[1].(map[string]any)["id"] != "timeline" {
		t.Fatalf("layout order not preserved: %v", saved)
	}
	if saved[1].(map[string]any)["background"] != "sage" {
		t.Fatalf("background not persisted: %v", saved[1])
	}

	status, body = call(t, admin, http.MethodPut, base+"/dashboard/layout", map[string]any{
		"layout": []map[string]any{{"id": "unknown", "visible": true}},
	})
	if status != http.StatusBadRequest {
		t.Fatalf("expected invalid layout error: %d %v", status, body)
	}

	status, body = call(t, admin, http.MethodPut, base+"/dashboard/layout", map[string]any{"reset": true})
	if status != http.StatusOK || body["layoutSource"] != "default" {
		t.Fatalf("reset layout: %d %v", status, body)
	}

	// 管理员发布家庭默认布局，不应写入个人副本。
	status, body = call(t, admin, http.MethodPut, base+"/household/default-layout", map[string]any{
		"layout": []map[string]any{
			{"id": "overview", "visible": true, "size": "wide", "background": "default"},
			{"id": "notes", "visible": true, "size": "wide", "background": "sand"},
		},
	})
	if status != http.StatusOK || body["layoutSource"] != "household" {
		t.Fatalf("publish household layout: %d %v", status, body)
	}
	status, body = call(t, admin, http.MethodGet, base+"/dashboard/layout", nil)
	if status != http.StatusOK || body["layoutSource"] != "household" {
		t.Fatalf("admin should follow household layout: %d %v", status, body)
	}
	if body["layout"].([]any)[1].(map[string]any)["id"] != "notes" {
		t.Fatalf("household layout not applied: %v", body["layout"])
	}

	member, _ := addMember(t, base, admin, "member1", "小林")
	status, body = call(t, member, http.MethodPut, base+"/household/default-layout", map[string]any{
		"layout": []map[string]any{{"id": "overview", "visible": true}},
	})
	if status != http.StatusForbidden {
		t.Fatalf("member should not publish layout: %d %v", status, body)
	}

	// 成员可保存个人布局覆盖家庭默认。
	status, body = call(t, member, http.MethodPut, base+"/dashboard/layout", map[string]any{
		"layout": []map[string]any{
			{"id": "overview", "visible": true, "size": "wide", "background": "default"},
			{"id": "messages", "visible": true, "size": "standard", "background": "sky"},
		},
	})
	if status != http.StatusOK || body["layoutSource"] != "personal" {
		t.Fatalf("member personal layout: %d %v", status, body)
	}
	status, body = call(t, member, http.MethodGet, base+"/dashboard", nil)
	if status != http.StatusOK || body["layoutSource"] != "personal" {
		t.Fatalf("member dashboard: %d %v", status, body)
	}
	memberLayout := body["layout"].([]any)
	if memberLayout[1].(map[string]any)["id"] != "messages" {
		t.Fatalf("member personal layout not applied: %v", memberLayout)
	}
}

func TestWeatherEndpoint(t *testing.T) {
	var upstreamHits int
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamHits++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"current":{"temperature_2m":18.2,"apparent_temperature":17.4,"weather_code":2},
"daily":{"time":["2026-11-24"],"weather_code":[2],"temperature_2m_max":[21.0],"temperature_2m_min":[12.0]}}`))
	}))
	defer upstream.Close()

	ts, admin := newTestServerWithConfig(t, func(cfg *config.Config) {
		cfg.WeatherBaseURL = upstream.URL
	})
	base := ts.URL + "/api/v1"
	if status, body := call(t, admin, http.MethodPost, base+"/setup", map[string]any{
		"username": "admin", "displayName": "管理员", "password": "password12345",
	}); status != http.StatusCreated {
		t.Fatalf("setup: %d %v", status, body)
	}

	status, body := call(t, admin, http.MethodGet, base+"/weather", nil)
	if status != http.StatusBadRequest {
		t.Fatalf("expected missing location error: %d %v", status, body)
	}

	if status, body = call(t, admin, http.MethodPatch, base+"/household", map[string]any{
		"defaultWeatherCity": "上海", "defaultWeatherLat": 31.23, "defaultWeatherLon": 121.47,
	}); status != http.StatusOK {
		t.Fatalf("set household weather: %d %v", status, body)
	}
	// 首页天气卡片只读缓存，不应因缺少缓存而请求上游。
	status, body = call(t, admin, http.MethodGet, base+"/dashboard/cards/weather", nil)
	if status != http.StatusOK || body["data"] != nil || body["error"] == nil {
		t.Fatalf("expected cache-only weather error: %d %v", status, body)
	}
	if upstreamHits != 0 {
		t.Fatalf("weather card should not hit upstream, got %d hits", upstreamHits)
	}

	status, body = call(t, admin, http.MethodGet, base+"/weather", nil)
	if status != http.StatusOK {
		t.Fatalf("weather: %d %v", status, body)
	}
	if body["city"] != "上海" || body["temperature"].(float64) != 18.2 || body["description"] != "多云" {
		t.Fatalf("unexpected weather payload: %v", body)
	}
	if upstreamHits != 1 {
		t.Fatalf("expected one upstream hit, got %d", upstreamHits)
	}

	// 写入缓存后，单卡片接口可直接返回数据。
	status, body = call(t, admin, http.MethodGet, base+"/dashboard/cards/weather", nil)
	if status != http.StatusOK || body["data"] == nil || body["error"] != nil {
		t.Fatalf("expected cached weather card data: %d %v", status, body)
	}
	if upstreamHits != 1 {
		t.Fatalf("expected cached card read, got %d hits", upstreamHits)
	}

	status, _ = call(t, admin, http.MethodGet, base+"/dashboard/cards/unknown", nil)
	if status != http.StatusBadRequest {
		t.Fatalf("expected unknown card error, got %d", status)
	}

	// 查询坐标优先，并且命中同一坐标缓存。
	status, body = call(t, admin, http.MethodGet, base+"/weather?lat=31.23&lon=121.47&city=上海", nil)
	if status != http.StatusOK || upstreamHits != 1 {
		t.Fatalf("expected cached coordinate hit: %d %v (hits=%d)", status, body, upstreamHits)
	}
}
