package weather

import (
	"context"
	"fmt"
	"strings"
	"time"
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
	cache     *diskCache
	now       func() time.Time
}

// NewManager 构造信息源管理器。def 为空或未知时使用 Open-Meteo。
// cacheDir 用于持久化天气数据，为空时只保留内存缓存。
func NewManager(openMeteo *Client, qweather *QWeather, def, cacheDir string) *Manager {
	m := &Manager{providers: map[string]Provider{}, def: def, cache: newDiskCache(cacheDir), now: time.Now}
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
// 上游出错时回退到持久化缓存，保证进程重启或接口卡住时仍能展示上次数据。
func (m *Manager) Current(ctx context.Context, preferred string, lat, lon float64, city string) (*Data, string, error) {
	p := m.Resolve(preferred)
	if p == nil {
		return nil, "", fmt.Errorf("no weather provider configured")
	}
	data, err := p.Current(ctx, lat, lon, city)
	if err != nil {
		if cached, source, ok := m.lookup(p.ID(), lat, lon); ok {
			cached.Stale = true
			if city != "" {
				cached.City = city
			}
			return cached, source, nil
		}
		return nil, p.ID(), err
	}
	data.Source = p.ID()
	key := cacheKey(p.ID(), lat, lon)
	if item, ok := m.cache.get(key); !ok || item.Data.UpdatedAt != data.UpdatedAt {
		m.cache.put(key, *data)
	}
	return data, p.ID(), nil
}

// Cached 只读取本地持久化缓存，不访问上游，避免天气阻塞首页其他卡片。
func (m *Manager) Cached(preferred string, lat, lon float64, city string) (*Data, bool) {
	data, _, ok := m.lookup(preferred, lat, lon)
	if !ok {
		return nil, false
	}
	if city != "" {
		data.City = city
	}
	return data, true
}

// lookup 按首选源 → 默认源 → 任一源的顺序查找缓存，并标记是否已过期。
func (m *Manager) lookup(preferred string, lat, lon float64) (*Data, string, bool) {
	for _, id := range m.candidates(preferred) {
		item, ok := m.cache.get(cacheKey(id, lat, lon))
		if !ok {
			continue
		}
		out := item.Data
		out.Source = id
		out.Stale = m.now().Sub(time.Unix(item.At, 0)) > cacheTTL
		return &out, id, true
	}
	return nil, "", false
}

func (m *Manager) candidates(preferred string) []string {
	out := make([]string, 0, len(m.order)+1)
	seen := map[string]bool{}
	add := func(id string) {
		if id == "" || seen[id] {
			return
		}
		if _, ok := m.providers[id]; !ok {
			return
		}
		seen[id] = true
		out = append(out, id)
	}
	add(preferred)
	add(m.def)
	for _, id := range m.order {
		add(id)
	}
	return out
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
