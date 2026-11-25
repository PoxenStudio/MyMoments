package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// QWeather 代理和风天气 API，并带内存缓存与降级。
type QWeather struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	ttl        time.Duration

	mu    sync.Mutex
	cache map[string]cacheItem
	now   func() time.Time
}

type qweatherNowResponse struct {
	Code string `json:"code"`
	Now  struct {
		Temp      string `json:"temp"`
		FeelsLike string `json:"feelsLike"`
		Text      string `json:"text"`
		Icon      string `json:"icon"`
	} `json:"now"`
}

type qweatherDailyResponse struct {
	Code  string `json:"code"`
	Daily []struct {
		FxDate  string `json:"fxDate"`
		TempMax string `json:"tempMax"`
		TempMin string `json:"tempMin"`
		TextDay string `json:"textDay"`
		IconDay string `json:"iconDay"`
	} `json:"daily"`
}

// NewQWeather 构造和风天气客户端。apiKey 为空时该信息源不可用。
func NewQWeather(baseURL, apiKey string, ttl time.Duration) *QWeather {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = "https://devapi.qweather.com/v7"
	}
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}
	return &QWeather{
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     strings.TrimSpace(apiKey),
		httpClient: &http.Client{Timeout: 5 * time.Second},
		ttl:        ttl,
		cache:      map[string]cacheItem{},
		now:        time.Now,
	}
}

// ID 实现 Provider 接口。
func (q *QWeather) ID() string { return SourceQWeather }

// Available 实现 Provider 接口。
func (q *QWeather) Available() bool { return q.apiKey != "" }

// Current 返回坐标处的天气，缓存策略与 Open-Meteo 客户端一致。
func (q *QWeather) Current(ctx context.Context, lat, lon float64, city string) (*Data, error) {
	if !q.Available() {
		return nil, ErrUnavailable
	}
	key := fmt.Sprintf("%.2f,%.2f", lat, lon)

	q.mu.Lock()
	cached, hasCache := q.cache[key]
	q.mu.Unlock()
	if hasCache && q.now().Sub(cached.at) < q.ttl {
		out := cached.data
		out.Stale = false
		if city != "" {
			out.City = city
		}
		return &out, nil
	}

	data, err := q.fetch(ctx, lat, lon)
	if err != nil {
		if hasCache {
			out := cached.data
			out.Stale = true
			if city != "" {
				out.City = city
			}
			return &out, nil
		}
		return nil, err
	}
	data.City = city
	if data.City == "" {
		data.City = fmt.Sprintf("%.2f, %.2f", lat, lon)
	}
	q.mu.Lock()
	q.cache[key] = cacheItem{data: *data, at: q.now()}
	q.mu.Unlock()
	return data, nil
}

func (q *QWeather) fetch(ctx context.Context, lat, lon float64) (*Data, error) {
	location := fmt.Sprintf("%.2f,%.2f", lon, lat)
	nowResponse := &qweatherNowResponse{}
	code, err := q.get(ctx, "/weather/now", location, nowResponse)
	if err != nil || code != "200" {
		return nil, ErrUnavailable
	}
	temp, _ := parseFloat(nowResponse.Now.Temp)
	apparent, _ := parseFloat(nowResponse.Now.FeelsLike)
	out := &Data{
		Temperature: temp,
		Apparent:    &apparent,
		Description: strings.TrimSpace(nowResponse.Now.Text),
		Code:        qweatherCode(nowResponse.Now.Icon),
		UpdatedAt:   q.now().Unix(),
	}
	if out.Description == "" {
		out.Description = Describe(out.Code)
	}

	dailyResponse := &qweatherDailyResponse{}
	dailyCode, err := q.get(ctx, "/weather/7d", location, dailyResponse)
	if err == nil && dailyCode == "200" && len(dailyResponse.Daily) > 0 {
		out.Forecast = make([]ForecastDay, 0, len(dailyResponse.Daily))
		for _, d := range dailyResponse.Daily {
			max, _ := parseFloat(d.TempMax)
			min, _ := parseFloat(d.TempMin)
			code := qweatherCode(d.IconDay)
			desc := strings.TrimSpace(d.TextDay)
			if desc == "" {
				desc = Describe(code)
			}
			out.Forecast = append(out.Forecast, ForecastDay{
				Date:        d.FxDate,
				Max:         max,
				Min:         min,
				Code:        code,
				Description: desc,
			})
		}
		high, low := out.Forecast[0].Max, out.Forecast[0].Min
		out.High = &high
		out.Low = &low
	}
	return out, nil
}

func (q *QWeather) get(ctx context.Context, path, location string, target any) (code string, err error) {
	values := url.Values{}
	values.Set("location", location)
	values.Set("key", q.apiKey)
	values.Set("lang", "zh")
	values.Set("unit", "m")
	endpoint := q.baseURL + path + "?" + values.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	res, err := q.httpClient.Do(req)
	if err != nil {
		return "", ErrUnavailable
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 4<<10))
		return "", ErrUnavailable
	}
	if err := json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(target); err != nil {
		return "", ErrUnavailable
	}
	switch payload := target.(type) {
	case *qweatherNowResponse:
		return payload.Code, nil
	case *qweatherDailyResponse:
		return payload.Code, nil
	}
	return "", ErrUnavailable
}

func parseFloat(raw string) (float64, bool) {
	v, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

// qweatherCode 将和风天气图标编码映射为前端使用的 WMO 天气代码。
func qweatherCode(icon string) int {
	icon = strings.TrimSpace(icon)
	switch icon {
	case "100", "150":
		return 0
	case "102", "103", "152", "153":
		return 1
	case "101", "151":
		return 2
	case "104", "154":
		return 3
	case "302", "303", "304":
		return 95
	case "313":
		return 66
	case "300", "305", "309":
		return 80
	case "301", "306", "307", "310", "311", "312", "314", "315", "316", "317", "318", "399":
		return 61
	}
	if icon >= "400" && icon <= "499" {
		return 71
	}
	if icon >= "500" && icon <= "515" {
		return 45
	}
	if icon == "900" {
		return 0
	}
	return 3
}
