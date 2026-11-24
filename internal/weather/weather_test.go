package weather

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

const samplePayload = `{"current":{"temperature_2m":21.5,"apparent_temperature":20.1,"weather_code":1},
"daily":{"time":["2026-11-24","2026-11-25"],"weather_code":[1,61],
"temperature_2m_max":[24.0,22.0],"temperature_2m_min":[15.0,14.0]}}`

func TestCurrentCachesAndFallsBackToStale(t *testing.T) {
	hits := 0
	fail := false
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if r.URL.Query().Get("latitude") == "" || r.URL.Query().Get("longitude") == "" {
			t.Errorf("missing coordinates: %s", r.URL.RawQuery)
		}
		if fail {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(samplePayload))
	}))
	defer upstream.Close()

	client := New(upstream.URL, "", time.Minute)
	now := time.Unix(1_800_000_000, 0)
	client.now = func() time.Time { return now }

	data, err := client.Current(context.Background(), 31.23, 121.47, "上海")
	if err != nil {
		t.Fatalf("current: %v", err)
	}
	if data.Temperature != 21.5 || data.City != "上海" || data.Description != "晴间多云" {
		t.Fatalf("unexpected data: %+v", data)
	}
	if data.High == nil || *data.High != 24 || data.Stale || len(data.Forecast) != 2 {
		t.Fatalf("unexpected summary: %+v", data)
	}
	if hits != 1 {
		t.Fatalf("expected one upstream hit, got %d", hits)
	}

	now = now.Add(30 * time.Second)
	if _, err := client.Current(context.Background(), 31.23, 121.47, "上海"); err != nil {
		t.Fatalf("cached current: %v", err)
	}
	if hits != 1 {
		t.Fatalf("expected cache hit, got %d upstream hits", hits)
	}

	now = now.Add(2 * time.Minute)
	fail = true
	stale, err := client.Current(context.Background(), 31.23, 121.47, "上海")
	if err != nil {
		t.Fatalf("stale fallback: %v", err)
	}
	if !stale.Stale || stale.Temperature != 21.5 {
		t.Fatalf("expected stale cache, got %+v", stale)
	}

	other, err := client.Current(context.Background(), 39.9, 116.4, "北京")
	if err == nil || other != nil {
		t.Fatalf("expected unavailable without cache, got %+v", other)
	}
}

func TestDescribe(t *testing.T) {
	cases := map[int]string{0: "晴", 3: "阴", 61: "下雨", 95: "雷阵雨", 999: "天气未知"}
	for code, want := range cases {
		if got := Describe(code); got != want {
			t.Fatalf("Describe(%d) = %q, want %q", code, got, want)
		}
	}
}
