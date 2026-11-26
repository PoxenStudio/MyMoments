package config

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Config 保存进程级配置，全部来自环境变量。
type Config struct {
	Addr               string
	DataDir            string
	LogDir             string
	DefaultTZ          string
	SessionSecret      []byte
	SessionTTL         time.Duration
	WeatherProvider    string
	WeatherAPIKey      string
	WeatherBaseURL     string
	QWeatherKey        string
	QWeatherBaseURL    string
	HolidayDataURL     string
	MaxImageBytes      int64
	MaxAudioBytes      int64
	MaxVideoBytes      int64
	HouseholdQuota     int64
	TrustedProxyHeader bool
	Dev                bool
}

// Load 读取环境变量并补齐可推导的默认值。
func Load() (Config, error) {
	c := Config{
		Addr:               env("MYMOMENTS_ADDR", ":8080"),
		DataDir:            env("MYMOMENTS_DATA_DIR", "./data"),
		DefaultTZ:          env("MYMOMENTS_DEFAULT_TZ", "Asia/Shanghai"),
		SessionTTL:         30 * 24 * time.Hour,
		WeatherProvider:    env("MYMOMENTS_WEATHER_PROVIDER", "open-meteo"),
		WeatherAPIKey:      env("MYMOMENTS_WEATHER_API_KEY", ""),
		WeatherBaseURL:     env("MYMOMENTS_WEATHER_BASE_URL", "https://api.open-meteo.com/v1"),
		QWeatherKey:        env("MYMOMENTS_QWEATHER_KEY", ""),
		QWeatherBaseURL:    env("MYMOMENTS_QWEATHER_BASE_URL", "https://devapi.qweather.com/v7"),
		HolidayDataURL:     env("MYMOMENTS_HOLIDAY_DATA_URL", "https://raw.githubusercontent.com/NateScarlet/holiday-cn/master/{year}.json"),
		MaxImageBytes:      20 << 20,
		MaxAudioBytes:      100 << 20,
		MaxVideoBytes:      200 << 20,
		HouseholdQuota:     1 << 30,
		TrustedProxyHeader: envBool("MYMOMENTS_TRUST_PROXY", false),
		Dev:                envBool("MYMOMENTS_DEV", false),
	}
	if v := os.Getenv("MYMOMENTS_HOUSEHOLD_QUOTA"); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil || n <= 0 {
			return Config{}, fmt.Errorf("invalid MYMOMENTS_HOUSEHOLD_QUOTA: %q", v)
		}
		c.HouseholdQuota = n
	}
	if _, err := time.LoadLocation(c.DefaultTZ); err != nil {
		return Config{}, fmt.Errorf("invalid MYMOMENTS_DEFAULT_TZ: %w", err)
	}
	if err := os.MkdirAll(c.DataDir, 0o750); err != nil {
		return Config{}, fmt.Errorf("create data dir: %w", err)
	}
	c.LogDir = env("MYMOMENTS_LOG_DIR", filepath.Join(c.DataDir, "logs"))
	if err := os.MkdirAll(filepath.Dir(c.DBPath()), 0o750); err != nil {
		return Config{}, fmt.Errorf("create db dir: %w", err)
	}
	secret, err := loadSecret(c.DataDir)
	if err != nil {
		return Config{}, err
	}
	c.SessionSecret = secret
	return c, nil
}

// DBPath 返回 SQLite 数据库文件路径，位于 data/db/moments.db。
func (c Config) DBPath() string {
	return filepath.Join(c.DataDir, "db", "moments.db")
}

// DSN 返回 SQLite 连接串，开启外键、WAL 与忙等待。
func (c Config) DSN() string {
	return "file:" + c.DBPath() + "?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=synchronous(NORMAL)"
}

// MediaDir 是受控媒体目录。
func (c Config) MediaDir() string { return filepath.Join(c.DataDir, "media") }

// PhotoAlbumDir 是首页照片墙的图片存储目录。
func (c Config) PhotoAlbumDir() string { return filepath.Join(c.DataDir, "photo_album") }

// HolidayDir 是法定节假日远程数据的缓存目录。
func (c Config) HolidayDir() string { return filepath.Join(c.DataDir, "holidays") }

func loadSecret(dataDir string) ([]byte, error) {
	if v := strings.TrimSpace(os.Getenv("MYMOMENTS_SESSION_SECRET")); v != "" {
		return []byte(v), nil
	}
	path := filepath.Join(dataDir, "session_secret")
	if b, err := os.ReadFile(path); err == nil {
		if len(strings.TrimSpace(string(b))) >= 32 {
			return []byte(strings.TrimSpace(string(b))), nil
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("read session secret: %w", err)
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return nil, fmt.Errorf("generate session secret: %w", err)
	}
	secret := []byte(hex.EncodeToString(buf))
	if err := os.WriteFile(path, secret, 0o600); err != nil {
		return nil, fmt.Errorf("persist session secret: %w", err)
	}
	return secret, nil
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	if v == "" {
		return fallback
	}
	return v == "1" || v == "true" || v == "yes" || v == "on"
}
