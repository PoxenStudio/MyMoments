package geoip

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ErrUnavailable 表示所有上游都无法识别出口 IP 的位置。
var ErrUnavailable = errors.New("geoip upstream unavailable")

const (
	sourceIPIP  = "myip.ipip.net"
	sourceIPWho = "ipwho.is"
)

// Result 是出口 IP 的位置信息；城市名原样来自上游，坐标可能缺失。
type Result struct {
	IP       string   `json:"ip,omitempty"`
	Country  string   `json:"country,omitempty"`
	Province string   `json:"province,omitempty"`
	City     string   `json:"city"`
	Lat      *float64 `json:"lat,omitempty"`
	Lon      *float64 `json:"lon,omitempty"`
	Source   string   `json:"source"`
}

// Lookup 通过公网出口 IP 反查大致位置，供浏览器定位不可用时兜底。
type Lookup struct {
	client   *http.Client
	ipipURL  string
	ipwhoURL string
}

// New 构造查询器。
func New(timeout time.Duration) *Lookup {
	if timeout <= 0 {
		timeout = 4 * time.Second
	}
	return &Lookup{
		client:   &http.Client{Timeout: timeout},
		ipipURL:  "https://myip.ipip.net/",
		ipwhoURL: "https://ipwho.is/",
	}
}

// Locate 依次尝试多个上游，返回第一个可用的结果。
func (l *Lookup) Locate(ctx context.Context) (Result, error) {
	var lastErr error
	for _, provider := range []func(context.Context) (Result, error){l.fromIPIP, l.fromIPWho} {
		res, err := provider(ctx)
		if err == nil {
			return res, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = ErrUnavailable
	}
	return Result{}, lastErr
}

func (l *Lookup) fromIPIP(ctx context.Context) (Result, error) {
	body, err := l.get(ctx, l.ipipURL)
	if err != nil {
		return Result{}, err
	}
	return parseIPIP(string(body))
}

func (l *Lookup) fromIPWho(ctx context.Context) (Result, error) {
	body, err := l.get(ctx, l.ipwhoURL)
	if err != nil {
		return Result{}, err
	}
	return parseIPWho(body)
}

func (l *Lookup) get(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "MyMoments/geoip")
	resp, err := l.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("geoip: %s returned %d", url, resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 64<<10))
}

// parseIPIP 解析 myip.ipip.net 的纯文本，例如：
// 当前 IP：1.2.3.4  来自于：中国 广东 深圳  电信
func parseIPIP(body string) (Result, error) {
	res := Result{Source: sourceIPIP}
	if _, ipPart, ok := strings.Cut(body, "IP："); ok {
		if fields := strings.Fields(ipPart); len(fields) > 0 {
			res.IP = fields[0]
		}
	}
	_, area, ok := strings.Cut(body, "来自于：")
	if !ok {
		_, area, ok = strings.Cut(body, "来自于:")
	}
	if !ok {
		return Result{}, ErrUnavailable
	}
	fields := strings.Fields(area)
	if len(fields) < 2 {
		return Result{}, ErrUnavailable
	}
	res.Country = fields[0]
	if len(fields) >= 3 {
		res.Province = fields[1]
		res.City = fields[2]
	} else {
		res.City = fields[1]
	}
	res.City = strings.TrimSpace(strings.TrimSuffix(res.City, "市"))
	if res.City == "" || res.City == "未知" || strings.Contains(res.City, "局域网") {
		return Result{}, ErrUnavailable
	}
	return res, nil
}

type ipwhoResponse struct {
	Success   bool     `json:"success"`
	Message   string   `json:"message"`
	IP        string   `json:"ip"`
	Country   string   `json:"country"`
	Region    string   `json:"region"`
	City      string   `json:"city"`
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`
}

// parseIPWho 解析 ipwho.is 的 JSON；城市通常是英文，坐标用于在本地城市表中就近匹配。
func parseIPWho(body []byte) (Result, error) {
	var payload ipwhoResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return Result{}, err
	}
	if !payload.Success {
		return Result{}, ErrUnavailable
	}
	if strings.TrimSpace(payload.City) == "" && payload.Latitude == nil {
		return Result{}, ErrUnavailable
	}
	return Result{
		IP:       payload.IP,
		Country:  payload.Country,
		Province: payload.Region,
		City:     strings.TrimSpace(payload.City),
		Lat:      payload.Latitude,
		Lon:      payload.Longitude,
		Source:   sourceIPWho,
	}, nil
}
