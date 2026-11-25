package store

import (
	"context"
	"path/filepath"
	"testing"
)

func TestOpen_MigratesAndIsReady(t *testing.T) {
	dir := t.TempDir()
	dsn := "file:" + filepath.Join(dir, "test.db") + "?_pragma=foreign_keys(1)"
	st, err := Open(context.Background(), dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer st.Close()

	var count int
	if err := st.DB.QueryRow(`SELECT COUNT(1) FROM sqlite_master WHERE type='table' AND name='users'`).Scan(&count); err != nil {
		t.Fatalf("query users table: %v", err)
	}
	if count != 1 {
		t.Fatalf("users table missing")
	}
	if err := st.IsReady(context.Background()); err != nil {
		t.Fatalf("ready: %v", err)
	}
}

func TestOpen_IsIdempotent(t *testing.T) {
	dir := t.TempDir()
	dsn := "file:" + filepath.Join(dir, "test.db")
	st, err := Open(context.Background(), dsn)
	if err != nil {
		t.Fatalf("first open: %v", err)
	}
	var first int
	if err := st.DB.QueryRow(`SELECT COUNT(1) FROM schema_migrations`).Scan(&first); err != nil {
		t.Fatalf("count migrations: %v", err)
	}
	st.Close()
	st2, err := Open(context.Background(), dsn)
	if err != nil {
		t.Fatalf("second open: %v", err)
	}
	defer st2.Close()
	var applied int
	if err := st2.DB.QueryRow(`SELECT COUNT(1) FROM schema_migrations`).Scan(&applied); err != nil {
		t.Fatalf("count migrations: %v", err)
	}
	if applied != first {
		t.Fatalf("expected %d migrations after reopen, got %d", first, applied)
	}
}

// TestOpen_ReappliesReplacedMigration 覆盖迁移文件被同版本号替换后的升级场景：
// 历史库记录了旧文件名，再次启动时应执行当前文件补齐缺失的结构。
func TestOpen_ReappliesReplacedMigration(t *testing.T) {
	dir := t.TempDir()
	dsn := "file:" + filepath.Join(dir, "test.db")
	st, err := Open(context.Background(), dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if _, err := st.DB.Exec(`ALTER TABLE household_settings DROP COLUMN default_weather_provider`); err != nil {
		t.Fatalf("drop column: %v", err)
	}
	if _, err := st.DB.Exec(`UPDATE schema_migrations SET name = '0002_task_reminders.sql' WHERE version = 2`); err != nil {
		t.Fatalf("rename migration: %v", err)
	}
	st.Close()

	st2, err := Open(context.Background(), dsn)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer st2.Close()
	var count int
	if err := st2.DB.QueryRow(`SELECT COUNT(1) FROM pragma_table_info('household_settings') WHERE name = 'default_weather_provider'`).Scan(&count); err != nil {
		t.Fatalf("check column: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected default_weather_provider restored")
	}
	var name string
	if err := st2.DB.QueryRow(`SELECT name FROM schema_migrations WHERE version = 2`).Scan(&name); err != nil {
		t.Fatalf("read migration name: %v", err)
	}
	if name != "0002_weather_provider.sql" {
		t.Fatalf("expected migration name updated, got %q", name)
	}
}
