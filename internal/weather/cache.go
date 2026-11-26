package weather

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// cacheTTL 是天气缓存被视为新鲜的有效期，与各信息源的内存缓存保持一致。
const cacheTTL = 10 * time.Minute

type cachedEntry struct {
	Data Data  `json:"data"`
	At   int64 `json:"at"`
}

// diskCache 把天气数据持久化到磁盘，供上游失败时回退，也可不访问上游直接读取。
type diskCache struct {
	dir   string
	mu    sync.RWMutex
	items map[string]cachedEntry
}

// newDiskCache 构造磁盘缓存，并加载已有文件，保证进程重启后仍能回退到上次数据。
func newDiskCache(dir string) *diskCache {
	c := &diskCache{dir: strings.TrimSpace(dir), items: map[string]cachedEntry{}}
	c.load()
	return c
}

func (c *diskCache) load() {
	if c.dir == "" {
		return
	}
	entries, err := os.ReadDir(c.dir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		body, err := os.ReadFile(filepath.Join(c.dir, entry.Name()))
		if err != nil {
			continue
		}
		var item cachedEntry
		if err := json.Unmarshal(body, &item); err != nil {
			continue
		}
		c.items[strings.TrimSuffix(entry.Name(), ".json")] = item
	}
}

func (c *diskCache) get(key string) (cachedEntry, bool) {
	c.mu.RLock()
	item, ok := c.items[key]
	c.mu.RUnlock()
	return item, ok
}

func (c *diskCache) put(key string, data Data) {
	item := cachedEntry{Data: data, At: time.Now().Unix()}
	c.mu.Lock()
	c.items[key] = item
	c.mu.Unlock()
	if c.dir == "" {
		return
	}
	if err := os.MkdirAll(c.dir, 0o750); err != nil {
		return
	}
	body, err := json.Marshal(item)
	if err != nil {
		return
	}
	final := filepath.Join(c.dir, key+".json")
	tmp := final + ".tmp"
	if err := os.WriteFile(tmp, body, 0o640); err != nil {
		return
	}
	if err := os.Rename(tmp, final); err != nil {
		_ = os.Remove(tmp)
	}
}

// cacheKey 生成按信息源与坐标区分的文件名，坐标保留两位小数与各信息源缓存粒度一致。
func cacheKey(providerID string, lat, lon float64) string {
	return fmt.Sprintf("%s_%s_%s", providerID, coordPart(lat), coordPart(lon))
}

func coordPart(v float64) string {
	return strings.ReplaceAll(fmt.Sprintf("%.2f", v), ".", "_")
}
