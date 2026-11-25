package weather

import (
	"context"
	"fmt"
	"strings"
)

// 内置天气信息源标识。
const (
	SourceOpenMeteo = "open-meteo"
	SourceQWeather  = "qweather"
)

// Provider 是天气信息源的统一接口。
type Provider interface {
	ID() string
	Available() bool
	Current(ctx context.Context, lat, lon float64, city string) (*Data, error)
}

// Source 描述一个可选天气信息源，供前端展示与选择。
type Source struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Note      string `json:"note"`
	Available bool   `json:"available"`
	Default   bool   `json:"default"`
}

// Manager 管理多个天气信息源，并负责在首选源不可用时回退到默认源。
type Manager struct {
	providers map[string]Provider
	order     []string
	def       string
}

// NewManager 构造信息源管理器。def 为空或未知时使用 Open-Meteo。
func NewManager(openMeteo *Client, qweather *QWeather, def string) *Manager {
	m := &Manager{providers: map[string]Provider{}, def: def}
	if openMeteo != nil {
		m.providers[SourceOpenMeteo] = openMeteo
		m.order = append(m.order, SourceOpenMeteo)
	}
	if qweather != nil {
		m.providers[SourceQWeather] = qweather
		m.order = append(m.order, SourceQWeather)
	}
	if _, ok := m.providers[m.def]; !ok {
		m.def = SourceOpenMeteo
	}
	return m
}

// Default 返回默认信息源标识。
func (m *Manager) Default() string { return m.def }

// Has 判断信息源是否已注册。
func (m *Manager) Has(id string) bool {
	_, ok := m.providers[id]
	return ok
}

// Available 判断信息源是否已配置可用。
func (m *Manager) Available(id string) bool {
	p, ok := m.providers[id]
	return ok && p.Available()
}

// Resolve 按首选源 → 默认源 → 任一可用源的顺序选择。
func (m *Manager) Resolve(preferred string) Provider {
	if p, ok := m.providers[preferred]; ok && p.Available() {
		return p
	}
	if p, ok := m.providers[m.def]; ok && p.Available() {
		return p
	}
	for _, id := range m.order {
		if p := m.providers[id]; p.Available() {
			return p
		}
	}
	return m.providers[SourceOpenMeteo]
}

// Current 使用解析后的信息源获取天气，并写入实际使用的信息源标识。
func (m *Manager) Current(ctx context.Context, preferred string, lat, lon float64, city string) (*Data, string, error) {
	p := m.Resolve(preferred)
	if p == nil {
		return nil, "", fmt.Errorf("no weather provider configured")
	}
	data, err := p.Current(ctx, lat, lon, city)
	if err != nil {
		return nil, p.ID(), err
	}
	data.Source = p.ID()
	return data, p.ID(), nil
}

// List 返回全部信息源描述。
func (m *Manager) List() []Source {
	out := make([]Source, 0, len(m.order))
	for _, id := range m.order {
		p := m.providers[id]
		item := Source{ID: id, Name: p.ID(), Available: p.Available(), Default: id == m.def}
		switch id {
		case SourceOpenMeteo:
			item.Name = "Open-Meteo"
			item.Note = "免费、无需密钥，默认信息源"
		case SourceQWeather:
			item.Name = "和风天气"
			item.Note = "需要服务端配置 MYMOMENTS_QWEATHER_KEY"
			if item.Available {
				item.Note = "已配置密钥，可切换使用"
			}
		default:
			item.Name = strings.ToUpper(id)
		}
		out = append(out, item)
	}
	return out
}
