package weather

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// ErrUnavailable 表示上游不可用且没有可用缓存。
var ErrUnavailable = errors.New("weather upstream unavailable")

// ForecastDay 是单日预报。
type ForecastDay struct {
	Date        string  `json:"date"`
	Max         float64 `json:"max"`
	Min         float64 `json:"min"`
	Code        int     `json:"code"`
	Description string  `json:"description"`
}

// Data 是天气摘要，序列化后直接返回给前端。
type Data struct {
	Source      string        `json:"source"`
	City        string        `json:"city"`
	Temperature float64       `json:"temperature"`
	Apparent    *float64      `json:"apparent"`
	Description string        `json:"description"`
	Code        int           `json:"code"`
	High        *float64      `json:"high"`
	Low         *float64      `json:"low"`
	UpdatedAt   int64         `json:"updatedAt"`
	Stale       bool          `json:"stale"`
	Forecast    []ForecastDay `json:"forecast"`
}

// ID 实现 Provider 接口。
func (c *Client) ID() string { return SourceOpenMeteo }

// Available 实现 Provider 接口；Open-Meteo 无需密钥，始终可用。
func (c *Client) Available() bool { return true }

// Client 代理 Open-Meteo，并带内存缓存与降级。
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	ttl        time.Duration

	mu    sync.Mutex
	cache map[string]cacheItem
	now   func() time.Time
}

type cacheItem struct {
	data Data
	at   time.Time
}

// New 构造天气客户端。baseURL 为空时使用 Open-Meteo 公共接口。
func New(baseURL, apiKey string, ttl time.Duration) *Client {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = "https://api.open-meteo.com/v1"
	}
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     strings.TrimSpace(apiKey),
		httpClient: &http.Client{Timeout: 5 * time.Second},
		ttl:        ttl,
		cache:      map[string]cacheItem{},
		now:        time.Now,
	}
}

type openMeteoResponse struct {
	Current struct {
		Temperature float64 `json:"temperature_2m"`
		Apparent    float64 `json:"apparent_temperature"`
		Code        int     `json:"weather_code"`
	} `json:"current"`
	Daily struct {
		Time []string  `json:"time"`
		Code []int     `json:"weather_code"`
		Max  []float64 `json:"temperature_2m_max"`
		Min  []float64 `json:"temperature_2m_min"`
	} `json:"daily"`
}

// Current 返回坐标处的天气。命中新鲜缓存直接返回；上游失败时回退到过期缓存并标记 stale。
func (c *Client) Current(ctx context.Context, lat, lon float64, city string) (*Data, error) {
	key := fmt.Sprintf("%.2f,%.2f", lat, lon)

	c.mu.Lock()
	cached, hasCache := c.cache[key]
	c.mu.Unlock()
	if hasCache && c.now().Sub(cached.at) < c.ttl {
		out := cached.data
		out.Stale = false
		if city != "" {
			out.City = city
		}
		return &out, nil
	}

	data, err := c.fetch(ctx, lat, lon)
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
	c.mu.Lock()
	c.cache[key] = cacheItem{data: *data, at: c.now()}
	c.mu.Unlock()
	return data, nil
}

func (c *Client) fetch(ctx context.Context, lat, lon float64) (*Data, error) {
	q := url.Values{}
	q.Set("latitude", fmt.Sprintf("%.4f", lat))
	q.Set("longitude", fmt.Sprintf("%.4f", lon))
	q.Set("current", "temperature_2m,apparent_temperature,weather_code")
	q.Set("daily", "weather_code,temperature_2m_max,temperature_2m_min")
	q.Set("timezone", "auto")
	q.Set("forecast_days", "4")
	if c.apiKey != "" {
		q.Set("apikey", c.apiKey)
	}
	endpoint := c.baseURL + "/forecast?" + q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, ErrUnavailable
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 4<<10))
		return nil, ErrUnavailable
	}
	var payload openMeteoResponse
	if err := json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&payload); err != nil {
		return nil, ErrUnavailable
	}
	if len(payload.Daily.Time) == 0 {
		return nil, ErrUnavailable
	}
	out := &Data{
		Temperature: payload.Current.Temperature,
		Code:        payload.Current.Code,
		Description: Describe(payload.Current.Code),
		UpdatedAt:   c.now().Unix(),
		Forecast:    make([]ForecastDay, 0, len(payload.Daily.Time)),
	}
	apparent := payload.Current.Apparent
	out.Apparent = &apparent
	for i, date := range payload.Daily.Time {
		day := ForecastDay{Date: date}
		if i < len(payload.Daily.Code) {
			day.Code = payload.Daily.Code[i]
			day.Description = Describe(day.Code)
		}
		if i < len(payload.Daily.Max) {
			day.Max = payload.Daily.Max[i]
		}
		if i < len(payload.Daily.Min) {
			day.Min = payload.Daily.Min[i]
		}
		out.Forecast = append(out.Forecast, day)
	}
	high, low := out.Forecast[0].Max, out.Forecast[0].Min
	out.High = &high
	out.Low = &low
	return out, nil
}

// Describe 把 WMO 天气代码翻译成中文描述。
func Describe(code int) string {
	switch code {
	case 0:
		return "晴"
	case 1:
		return "晴间多云"
	case 2:
		return "多云"
	case 3:
		return "阴"
	case 45, 48:
		return "有雾"
	case 51, 53, 55:
		return "毛毛雨"
	case 56, 57:
		return "冻毛毛雨"
	case 61, 63, 65:
		return "下雨"
	case 66, 67:
		return "冻雨"
	case 71, 73, 75:
		return "下雪"
	case 77:
		return "米雪"
	case 80, 81, 82:
		return "阵雨"
	case 85, 86:
		return "阵雪"
	case 95:
		return "雷阵雨"
	case 96, 99:
		return "雷阵雨伴冰雹"
	default:
		return "天气未知"
	}
}
