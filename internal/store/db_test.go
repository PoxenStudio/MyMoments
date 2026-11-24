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
