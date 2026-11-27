package httpapi

import (
	"sync"
	"time"
)

type attemptWindow struct {
	failures []int64
	lastSeen int64
}

// loginLimiter 对登录失败做进程内限流；单实例部署足够。
type loginLimiter struct {
	mu      sync.Mutex
	windows map[string]*attemptWindow
	limit   int
	window  time.Duration
}

func newLoginLimiter(limit int, window time.Duration) *loginLimiter {
	return &loginLimiter{windows: make(map[string]*attemptWindow), limit: limit, window: window}
}

func (l *loginLimiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now().Unix()
	w := l.windows[key]
	if w == nil {
		return true
	}
	cutoff := now - int64(l.window.Seconds())
	kept := w.failures[:0]
	for _, ts := range w.failures {
		if ts >= cutoff {
			kept = append(kept, ts)
		}
	}
	w.failures = kept
	return len(w.failures) < l.limit
}

func (l *loginLimiter) fail(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now().Unix()
	w := l.windows[key]
	if w == nil {
		w = &attemptWindow{}
		l.windows[key] = w
	}
	w.failures = append(w.failures, now)
	w.lastSeen = now
	if len(l.windows) > 2048 {
		l.cleanupLocked(now)
	}
}

func (l *loginLimiter) success(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.windows, key)
}

func (l *loginLimiter) cleanupLocked(now int64) {
	cutoff := now - int64(l.window.Seconds())*2
	for k, w := range l.windows {
		if w.lastSeen < cutoff {
			delete(l.windows, k)
		}
	}
}
